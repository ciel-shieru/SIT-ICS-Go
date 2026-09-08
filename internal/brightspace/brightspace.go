package brightspace

import (
	"fmt"
	"strings"
	"time"
)

// calendarEventToEntry converts a CalendarEventAPI to a BrightSpaceEntry with parsed timestamps.
func calendarEventToEntry(ev CalendarEventAPI, source SourceType, orgUnitID, orgUnitName, orgUnitCode string, loc *time.Location) BrightSpaceEntry {
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

	dtStart := parseStringTimestamp(ev.StartDateTime, loc)
	dtEnd := parseStringTimestamp(ev.EndDateTime, loc)

	return BrightSpaceEntry{
		Source:      source,
		Title:       title,
		OrgUnitId:   orgUnitID,
		OrgUnitName: orgUnitName,
		OrgUnitCode: orgUnitCode,
		Location:    ev.LocationName,
		Description: strings.Join(descParts, "\n"),
		DTStart:     dtStart,
		DTEnd:       dtEnd,
		IsAllDay:    ev.IsAllDayEvent,
	}
}

// dropboxFolderToEntry converts a DropboxFolderAPI to a BrightSpaceEntry with parsed timestamps.
func dropboxFolderToEntry(folder DropboxFolderAPI, source SourceType, orgUnitID, orgUnitName, orgUnitCode string, loc *time.Location) BrightSpaceEntry {
	title := fmt.Sprintf("[%s] %s", "Submission Due", folder.Name)
	descParts := []string{fmt.Sprintf("Dropbox: %s", folder.Name)}

	dtStart := parseStringTimestamp(folder.DueDate, loc)

	return BrightSpaceEntry{
		Source:      source,
		Title:       title,
		OrgUnitId:   orgUnitID,
		OrgUnitName: orgUnitName,
		OrgUnitCode: orgUnitCode,
		Description: strings.Join(descParts, "\n"),
		DTStart:     dtStart,
		DTEnd:       dtStart,
		IsAllDay:    false,
	}
}

// parseStringTimestamp parses a timestamp string into time.Time.
func parseStringTimestamp(s string, loc *time.Location) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err2 := time.Parse("2006-01-02T15:04:05.000Z", s)
		if err2 != nil {
			return time.Time{}
		}
		t = t.In(loc)
	}
	if loc != nil {
		t = t.In(loc)
	}
	return t
}
