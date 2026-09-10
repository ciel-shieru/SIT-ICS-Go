package brightspace

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

// APIToStringEntry converts a CalendarEventAPI to a BrightSpaceStringEntry.
func APIToStringEntry(ev CalendarEventAPI, source string) BrightSpaceStringEntry {
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

	return BrightSpaceStringEntry{
		Title:       title,
		OrgUnitId:   fmt.Sprintf("%d", ev.OrgUnitId),
		OrgUnitName: ev.OrgUnitName,
		OrgUnitCode: ev.OrgUnitCode,
		Location:    ev.LocationName,
		Description: strings.Join(descParts, "\n"),
		DTStart:     ev.StartDateTime,
		DTEnd:       ev.EndDateTime,
		IsAllDay:    ev.IsAllDayEvent,
		Source:      source,
	}
}

// FolderToStringEntry converts a DropboxFolderAPI to a BrightSpaceStringEntry.
func FolderToStringEntry(folder DropboxFolderAPI) BrightSpaceStringEntry {
	return BrightSpaceStringEntry{
		Title:       fmt.Sprintf("[Submission Due] %s", folder.Name),
		OrgUnitId:   folder.OrgUnitId,
		OrgUnitName: folder.OrgUnitName,
		OrgUnitCode: folder.OrgUnitCode,
		Description: "Dropbox: " + folder.Name,
		DTStart:     folder.DueDate,
		DTEnd:       folder.DueDate,
		IsAllDay:    false,
		Source:      "brightspace-dropbox",
	}
}



// htmlToPlainText strips HTML tags and common entities from an HTML string.
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
	return strings.TrimSpace(text)
}

// BrightSpaceStringEntry represents a BrightSpace event with string-based timestamps,
// as returned by the browser scraping layer.
type BrightSpaceStringEntry struct {
	Title       string
	OrgUnitId   string
	OrgUnitName string
	OrgUnitCode string
	Location    string
	Description string
	DTStart     string
	DTEnd       string
	IsAllDay    bool
	Source      string
}

// ParseTimestamp parses a BrightSpace timestamp string into a time.Time value.
// It tries RFC3339Nano first, then falls back to "2006-01-02T15:04:05.000Z".
func ParseTimestamp(s string, loc *time.Location) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err2 := time.Parse("2006-01-02T15:04:05.000Z", s)
		if err2 != nil {
			return time.Time{}, fmt.Errorf("parse %q: not RFC3339 or YYYY-MM-DDTHH:MM:SS.sssZ", s)
		}
		t = t.In(loc)
	}
	if loc != nil {
		t = t.In(loc)
	}
	return t, nil
}

// EntriesToEvents converts BrightSpace string entries to calendar.Event values,
// applying blocklist filtering.
func EntriesToEvents(entries []BrightSpaceStringEntry, blocklist *Blocklist, loc *time.Location) []calendar.Event {
	events := make([]calendar.Event, 0, len(entries))
	for _, entry := range entries {
		if blocklist.IsCourseBlocked(entry.OrgUnitId, entry.OrgUnitName) {
			log.Printf("brightspace: blocked course %s (%s)", entry.OrgUnitName, entry.OrgUnitId)
			continue
		}
		if blocklist.IsEventBlocked(entry.Title) {
			log.Printf("brightspace: blocked event %q in %s", entry.Title, entry.OrgUnitName)
			continue
		}
		if blocklist.IsLocationBlocked(entry.Location) {
			log.Printf("brightspace: blocked event %q in %s (location %q)", entry.Title, entry.OrgUnitName, entry.Location)
			continue
		}

		dtStart, err := ParseTimestamp(entry.DTStart, loc)
		if err != nil {
			log.Printf("brightspace: skipping entry %q: invalid start time %q: %v", entry.Title, entry.DTStart, err)
			continue
		}

		dtEnd := dtStart
		if entry.DTEnd != "" {
			endTime, err := ParseTimestamp(entry.DTEnd, loc)
			if err == nil {
				dtEnd = endTime
			}
		}

		if entry.IsAllDay {
			dtEnd = dtStart.Add(24 * time.Hour)
		}

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

		events = append(events, calendar.Event{
			DTStart:     dtStart,
			DTEnd:       dtEnd,
			Summary:     summary,
			Title:       entry.Title,
			OrgUnitID:   entry.OrgUnitId,
			OrgUnitName: entry.OrgUnitName,
			OrgUnitCode: entry.OrgUnitCode,
			Location:    entry.Location,
			Description: entry.Description,
			Source:      entry.Source,
		})
	}
	return events
}
