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
)

func runFetch(cfg *config.Config, provider *auth.ADFSProvider, cache *calendar.ICSCache, loc *time.Location) {
	log.Printf("scheduler: starting timetable fetch")
	ctx, cancel := context.WithTimeout(context.Background(), FetchTimeout)
	defer cancel()

	authResult, err := provider.Authenticate(ctx, auth.AuthRequest{
		Username:      cfg.Username,
		Password:      cfg.Password,
		TOTPSecret:    cfg.TOTPSecret,
		PeopleSoftURL: "https://in4sit.singaporetech.edu.sg/",
	})
	if err != nil {
		log.Printf("scheduler: auth failed: %v", err)
		return
	}

	log.Printf("scheduler: auth successful, %d cookies set, fetching timetable", len(authResult.Cookies))

	entries, err := provider.FetchTimetable(ctx, "")
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

	if cfg.BrightSpaceEnabled {
		blocklist := &brightspace.Blocklist{
			CourseNamePatterns:    brightspace.ParseCommaSeparated(cfg.BrightSpaceCourseNameBlocklist),
			CourseIDs:             brightspace.ParseCommaSeparated(cfg.BrightSpaceCourseIDBlocklist),
			EventTitlePatterns:    brightspace.ParseCommaSeparated(cfg.BrightSpaceEventTitleBlocklist),
			EventLocationPatterns: brightspace.ParseCommaSeparated(cfg.BrightSpaceEventLocationBlocklist),
		}

		deleted := cache.RemoveWhere(func(e calendar.Event) bool {
			return blocklist.Matches(e.OrgUnitID, e.OrgUnitName, e.Title, e.Location)
		})
		if deleted > 0 {
			log.Printf("scheduler: deleted %d blocked brightspace events from cache", deleted)
		}

		bsEntries, err := provider.FetchBrightSpace(ctx, cfg.BrightSpaceBaseURL)
		if err != nil {
			log.Printf("scheduler: brightspace fetch failed: %v", err)
		} else {
			var bsStringEntries []brightspace.BrightSpaceStringEntry
			for _, e := range bsEntries {
				bsStringEntries = append(bsStringEntries, brightspace.BrightSpaceStringEntry{
					Title:       e.Title,
					OrgUnitId:   e.OrgUnitId,
					OrgUnitName: e.OrgUnitName,
					OrgUnitCode: e.OrgUnitCode,
					Location:    e.Location,
					Description: e.Description,
					DTStart:     e.DTStart,
					DTEnd:       e.DTEnd,
					IsAllDay:    e.IsAllDay,
					Source:      e.Source,
				})
			}
			bsEvents := brightspace.EntriesToEvents(bsStringEntries, blocklist, loc)
			icsEvents = append(icsEvents, bsEvents...)
			log.Printf("scheduler: added %d brightspace events", len(bsEvents))
		}
	}

	if err := cache.Update(icsEvents, cfg.TZ); err != nil {
		log.Printf("scheduler: update failed: %v", err)
		return
	}

	if err := saveAll(cache, cfg.ICSStoragePath, cfg.ICSOnlinePath, cfg.ICSCampusPath, cfg.XsiteEventsPath, cfg.XsiteDropboxPath, cfg.TZ, cfg.ICSRefreshInterval); err != nil {
		log.Printf("scheduler: save failed: %v", err)
		return
	}

	log.Printf("scheduler: fetch complete, %d events cached", len(icsEvents))
}
