package brightspace

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/ics"
)

// Fetch extracts BrightSpace D2L calendar events and dropbox due dates,
// returning them as ics.Event entries ready for the ICS cache.
func Fetch(ctx context.Context, client *Client, cfg *FetchConfig, loc *time.Location) ([]ics.Event, error) {
	blocklist := &Blocklist{
		CourseNamePatterns: cfg.CourseNameBlocklist,
		CourseIDs:          cfg.CourseIDBlocklist,
		EventTitlePatterns: cfg.EventTitleBlocklist,
	}

	version, err := client.CheckVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("check version: %w", err)
	}
	log.Printf("brightspace: API version %s", version)

	courses, err := client.FetchCourses(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch courses: %w", err)
	}
	log.Printf("brightspace: found %d courses", len(courses))

	var icsEvents []ics.Event
	var calendarEvents []ics.Event
	var dropboxEvents []ics.Event

	for _, course := range courses {
		if blocklist.IsCourseBlocked(course.OrgUnitId, course.Name) {
			log.Printf("brightspace: blocked course %s (%s)", course.Name, course.OrgUnitId)
			continue
		}

		// Fetch calendar events
		events, err := client.FetchCalendarEvents(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch calendar events for %s: %v", course.Name, err)
			continue
		}
		for _, ev := range events {
			entry := calendarEventToEntry(ev, SourceCalendar, strconv.Itoa(ev.OrgUnitId), course.Name, course.Code)
			if entry.Title == "" {
				continue
			}
			if blocklist.IsEventBlocked(entry.Title) {
				log.Printf("brightspace: blocked event %q in %s", entry.Title, course.Name)
				continue
			}
			evEvent := entryToICSEvent(&entry, loc)
			if evEvent != nil {
				calendarEvents = append(calendarEvents, *evEvent)
				icsEvents = append(icsEvents, *evEvent)
			}
		}

		// Fetch dropbox folders
		folders, err := client.FetchDropboxFolders(ctx, version, course.OrgUnitId)
		if err != nil {
			log.Printf("brightspace: failed to fetch dropbox folders for %s: %v", course.Name, err)
			continue
		}
		for _, folder := range folders {
			entry := dropboxFolderToEntry(folder, SourceDropbox, course.OrgUnitId, course.Name, course.Code)
			if entry.Title == "" {
				continue
			}
			if blocklist.IsEventBlocked(entry.Title) {
				log.Printf("brightspace: blocked dropbox %q in %s", entry.Title, course.Name)
				continue
			}
			evEvent := entryToICSEvent(&entry, loc)
			if evEvent != nil {
				dropboxEvents = append(dropboxEvents, *evEvent)
				icsEvents = append(icsEvents, *evEvent)
			}
		}
	}

	log.Printf("brightspace: extracted %d calendar events, %d dropbox due dates", len(calendarEvents), len(dropboxEvents))
	return icsEvents, nil
}

// FetchConfig holds the configuration for a BrightSpace fetch.
type FetchConfig struct {
	CourseNameBlocklist []string
	CourseIDBlocklist   []string
	EventTitleBlocklist []string
}

func calendarEventToEntry(ev CalendarEvent, source SourceType, orgUnitID, orgUnitName, orgUnitCode string) BrightSpaceEntry {
	title := ev.Title
	if title == "" {
		title = ev.OrgUnitName
	}

	descParts := []string{}
	if ev.Description != "" {
		plainDesc := htmlToPlainText(ev.Description)
		if plainDesc != "" {
			descParts = append(descParts, plainDesc)
		}
	}
	if ev.LocationName != "" {
		descParts = append(descParts, "Location: "+ev.LocationName)
	}

	return BrightSpaceEntry{
		Source:      source,
		Title:       title,
		OrgUnitId:   orgUnitID,
		OrgUnitName: orgUnitName,
		OrgUnitCode: orgUnitCode,
		Location:    ev.LocationName,
		Description: strings.Join(descParts, "\n"),
		DTStart:     ev.StartDateTime,
		DTEnd:       ev.EndDateTime,
		IsAllDay:    ev.IsAllDayEvent,
	}
}

func dropboxFolderToEntry(folder DropboxFolder, source SourceType, orgUnitID, orgUnitName, orgUnitCode string) BrightSpaceEntry {
	title := fmt.Sprintf("[%s] %s", "Submission Due", folder.Name)
	descParts := []string{fmt.Sprintf("Dropbox: %s", folder.Name)}

	return BrightSpaceEntry{
		Source:      source,
		Title:       title,
		OrgUnitId:   orgUnitID,
		OrgUnitName: orgUnitName,
		OrgUnitCode: orgUnitCode,
		Description: strings.Join(descParts, "\n"),
		DTStart:     folder.DueDate,
		DTEnd:       folder.DueDate,
		IsAllDay:    false,
	}
}

func entryToICSEvent(entry *BrightSpaceEntry, loc *time.Location) *ics.Event {
	if entry == nil {
		return nil
	}

	dtStart := entry.DTStart
	dtEnd := entry.DTEnd

	if !dtStart.IsZero() && loc != nil {
		dtStart = dtStart.In(loc)
	}
	if !dtEnd.IsZero() && loc != nil {
		dtEnd = dtEnd.In(loc)
	}

	// For all-day events, set a full day duration
	if entry.IsAllDay {
		dtEnd = dtStart.Add(24 * time.Hour)
	}

	// Skip events with zero times
	if dtStart.IsZero() {
		return nil
	}

	// For dropbox due dates with zero end time, set end to start + 1 day (all-day style)
	if entry.Source == SourceDropbox && dtEnd.IsZero() {
		dtEnd = dtStart.Add(24 * time.Hour)
	}

	// Shorten org unit name for summary if it's very long
	orgUnitName := entry.OrgUnitName
	if len(orgUnitName) > 80 {
		orgUnitName = orgUnitName[:77] + "..."
	}

	summary := entry.Title
	if entry.OrgUnitCode != "" {
		summary = fmt.Sprintf("[%s] %s", entry.OrgUnitCode, entry.Title)
	} else if orgUnitName != "" {
		summary = fmt.Sprintf("[%s] %s", orgUnitName, entry.Title)
	}

	return &ics.Event{
		DTStart:     dtStart,
		DTEnd:       dtEnd,
		Summary:     summary,
		Location:    entry.Location,
		Description: entry.Description,
	}
}

func htmlToPlainText(html string) string {
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br />", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n")
	html = strings.ReplaceAll(html, "<p>", "")

	var result strings.Builder
	inTag := false
	for _, ch := range html {
		switch ch {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(ch)
			}
		}
	}

	text := result.String()
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.TrimSpace(text)

	return text
}
