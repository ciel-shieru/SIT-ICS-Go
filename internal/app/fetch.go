package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
	"github.com/ciel-shieru/sit-ics-go/internal/smartmerge"
)

func runFetch(ctx context.Context, cfg *config.Config, provider *auth.ADFSProvider, cache *calendar.ICSCache, loc *time.Location) {
	log.Printf("scheduler: starting timetable fetch")

	if ctx.Err() != nil {
		log.Printf("scheduler: fetch cancelled: %v", ctx.Err())
		return
	}

	// Authentication is its own operation. Do not reuse its context for the
	// subsequent PeopleSoft timetable fetch: a slow MFA/SAML flow can consume
	// most or all of the deadline before the timetable page is even opened.
	authCtx, authCancel := context.WithTimeout(ctx, FetchTimeout)
	started := time.Now()
	_, err := provider.Authenticate(authCtx, auth.AuthRequest{
		Username:      cfg.Username,
		Password:      cfg.Password,
		TOTPSecret:    cfg.TOTPSecret,
		PeopleSoftURL: "https://in4sit.singaporetech.edu.sg/",
	})
	authCancel()

	if err != nil {
		log.Printf("scheduler: auth failed: %v", err)
		return
	}

	log.Printf("scheduler: auth successful after %s, fetching timetable", time.Since(started).Round(time.Millisecond))

	// Start a fresh deadline for timetable fetching. This is the critical fix:
	// the PeopleSoft page must not inherit an authentication deadline that may
	// already be expired or nearly expired.
	timetableCtx, timetableCancel := context.WithTimeout(ctx, TimetableFetchTimeout)
	entries, err := provider.FetchTimetable(timetableCtx, "", loc)
	timetableCancel()
	if err != nil {
		log.Printf("scheduler: fetch failed: %v", err)
		return
	}
	allEntries := entries

	icsEvents := make([]calendar.Event, 0, len(allEntries))
	for _, entry := range allEntries {
		dtStart, dtEnd, err := peoplesoft.ParseEntryDateTime(entry, loc)
		if err != nil {
			log.Printf("scheduler: skipping entry %s: %v", entry.CourseCode, err)
			continue
		}
		icsEvents = append(icsEvents, calendar.Event{
			CourseCode:  entry.CourseCode,
			Summary:     fmt.Sprintf("%s - %s (%s)", entry.CourseCode, entry.Section, entry.Type),
			Location:    entry.Location,
			Description: fmt.Sprintf("Course: %s\nClass: %s\nSection: %s\nType: %s", entry.CourseCode, entry.ClassName, entry.Section, entry.Type),
			DTStart:     dtStart,
			DTEnd:       dtEnd,
		})
	}

	if cfg.XsiteEnabled {
		blocklist := &brightspace.Blocklist{
			CourseNamePatterns:    brightspace.ParseCommaSeparated(cfg.XsiteCourseNameBlocklist),
			CourseIDs:             brightspace.ParseCommaSeparated(cfg.XsiteCourseIDBlocklist),
			CourseCodePatterns:    brightspace.ParseCommaSeparated(cfg.XsiteCourseCodeBlocklist),
			EventTitlePatterns:    brightspace.ParseCommaSeparated(cfg.XsiteEventTitleBlocklist),
			EventLocationPatterns: brightspace.ParseCommaSeparated(cfg.XsiteEventLocationBlocklist),
			QuizTitlePatterns:     brightspace.ParseCommaSeparated(cfg.XsiteQuizTitleBlocklist),
		}
		blocklist.CompilePatterns()

		deleted := cache.RemoveWhere(func(e calendar.Event) bool {
			return blocklist.Matches(e.OrgUnitID, e.OrgUnitName, e.OrgUnitCode, e.Title, e.Location)
		})
		if deleted > 0 {
			log.Printf("scheduler: deleted %d blocked brightspace events from cache", deleted)
		}

		brightSpaceCtx, brightSpaceCancel := context.WithTimeout(ctx, BrightSpaceFetchTimeout)
		bsEntries, err := provider.FetchBrightSpace(brightSpaceCtx, "https://xsite.singaporetech.edu.sg")
		brightSpaceCancel()
		var bsEvents []calendar.Event
		if err != nil {
			log.Printf("scheduler: brightspace fetch failed: %v", err)
		} else {
			bsEvents = brightspace.EntriesToEvents(bsEntries, blocklist, loc)
			if cfg.XsiteSmartMergeEnabled {
				psEvents, mergedCount, matchedCalendarEventIDs := smartmerge.MergeEvents(icsEvents, bsEvents)
				if mergedCount > 0 {
					log.Printf("smartmerge: merged %d brightspace events into timetable events", mergedCount)
					removed := cache.RemoveWhere(func(e calendar.Event) bool {
						return matchedCalendarEventIDs[e.CalendarEventID]
					})
					if removed > 0 {
						log.Printf("smartmerge: removed %d matched brightspace calendar events from cache", removed)
					}
				} else {
					log.Printf("smartmerge: no matching events found (checked %d brightspace events)", len(bsEvents))
				}
				icsEvents = psEvents
			} else {
				icsEvents = append(icsEvents, bsEvents...)
				log.Printf("scheduler: smart merge disabled, added %d brightspace events", len(bsEvents))
			}
		}

		brightSpaceQuizzesCtx, brightSpaceQuizzesCancel := context.WithTimeout(ctx, BrightSpaceFetchTimeout)
		bsQuizzesAPI, err := provider.FetchBrightSpaceQuizzesAPI(brightSpaceQuizzesCtx, "https://xsite.singaporetech.edu.sg")
		brightSpaceQuizzesCancel()
		var bsQuizzesEntries []brightspace.BrightSpaceStringEntry
		if err != nil {
			log.Printf("scheduler: brightspace quizzes fetch failed: %v", err)
		} else {
			for _, q := range bsQuizzesAPI {
				bsQuizzesEntries = append(bsQuizzesEntries, brightspace.QuizToStringEntry(q))
			}

			bsQuizzes := brightspace.QuizEntriesToEvents(bsQuizzesEntries, blocklist, loc)

			bsMerged, replacedCount, removedCalendarEventIDs := smartmerge.DedupQuizzes(icsEvents, bsQuizzes)
			if replacedCount > 0 {
				log.Printf("smartmerge: replaced %d quiz-associated calendar events with quiz entries", replacedCount)
				removed := cache.RemoveWhere(func(e calendar.Event) bool {
					return removedCalendarEventIDs[e.CalendarEventID]
				})
				if removed > 0 {
					log.Printf("smartmerge: removed %d quiz-associated calendar events from cache", removed)
				}
			}

			icsEvents = bsMerged
			log.Printf("scheduler: added %d brightspace quiz events", len(bsMerged))

			if cfg.XsiteQuizAttemptTrackingEnabled {
				removedQuizIDs := make(map[int]bool)

				for _, q := range bsQuizzesAPI {
					if q.QuizId == 0 {
						continue
					}

					quizURL := fmt.Sprintf(
						"https://xsite.singaporetech.edu.sg/d2l/lms/quizzing/user/quiz_submissions.d2l?qi=%d&ou=%s",
						q.QuizId, q.OrgUnitId,
					)

					quizCtx, quizCancel := context.WithTimeout(ctx, BrightSpaceFetchTimeout)
					pageHTML, fetchErr := provider.FetchQuizSubmissionPage(quizCtx, quizURL)
					quizCancel()
					if fetchErr != nil {
						log.Printf("scheduler: failed to fetch quiz submission page for quiz %d: %v", q.QuizId, fetchErr)
						continue
					}

					attemptInfo, parseErr := brightspace.ParseQuizSubmissionHTML(pageHTML, q.QuizId, q.OrgUnitId)
					if parseErr != nil {
						log.Printf("scheduler: failed to parse quiz submission for quiz %d: %v", q.QuizId, parseErr)
						continue
					}

					if brightspace.ShouldRemoveQuiz(
						q.AttemptsAllowed.IsUnlimited,
						q.AttemptsAllowed.NumberOfAttemptsAllowed,
						attemptInfo.AttemptsMade,
						attemptInfo.BestScore,
					) {
						log.Printf("scheduler: removing quiz %q (ID=%d) — attempts=%d, best=%.1f%%, unlimited=%v",
							q.Name, q.QuizId, attemptInfo.AttemptsMade, attemptInfo.BestScore, q.AttemptsAllowed.IsUnlimited)
						removedQuizIDs[q.QuizId] = true
					}
				}

				if len(removedQuizIDs) > 0 {
					newEvents := make([]calendar.Event, 0, len(icsEvents))
					for _, e := range icsEvents {
						if e.Source != "brightspace-quizzes" || !removedQuizIDs[e.QuizID] {
							newEvents = append(newEvents, e)
						}
					}
					icsEvents = newEvents
					log.Printf("scheduler: removed %d completed quizzes from ICS", len(removedQuizIDs))

					removed := cache.RemoveWhere(func(e calendar.Event) bool {
						return e.Source == "brightspace-quizzes" && removedQuizIDs[e.QuizID]
					})
					if removed > 0 {
						log.Printf("scheduler: removed %d completed quizzes from cache", removed)
					}
				}
			}
		}
	}

	if err := cache.Update(icsEvents, cfg.TZ); err != nil {
		log.Printf("scheduler: update failed: %v", err)
		return
	}

	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts := calendar.AlertsFromConfig(
		cfg.TimetableAlerts,
		cfg.TimetableOnlineAlerts,
		cfg.TimetableCampusAlerts,
		cfg.XsiteEventsAlerts,
		cfg.XsiteDropboxAlerts,
		cfg.XsiteQuizzesAlerts,
	)

	if err := saveAllOutputs(cache, cfg.ICSStoragePath, cfg.ICSOnlinePath, cfg.ICSCampusPath, cfg.XsiteEventsPath, cfg.XsiteDropboxPath, cfg.XsiteQuizzesPath, cfg.XsitePath, cfg.TZ, cfg.ICSRefreshInterval, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts); err != nil {
		log.Printf("scheduler: save failed: %v", err)
		return
	}

	log.Printf("scheduler: fetch complete, %d events cached", len(icsEvents))
}
