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
		if strings.Contains(strings.ToLower(ev.Description), "zoom.us") {
			descParts = append(descParts, ev.Description)
		} else {
			plainDesc := htmlToPlainText(ev.Description)
			if plainDesc != "" {
				descParts = append(descParts, plainDesc)
			}
		}
	}
	if ev.LocationName != "" {
		descParts = append(descParts, "Location: "+ev.LocationName)
	}

	var quizID int
	if ev.AssociatedEntity != nil && ev.AssociatedEntity.AssociatedEntityType == "D2L.LE.Quizzing.Quiz" {
		quizID = ev.AssociatedEntity.AssociatedEntityId
	}

	return BrightSpaceStringEntry{
		Title:                 title,
		OrgUnitId:             fmt.Sprintf("%d", ev.OrgUnitId),
		OrgUnitName:           ev.OrgUnitName,
		OrgUnitCode:           ev.OrgUnitCode,
		Location:              ev.LocationName,
		Description:           strings.Join(descParts, "\n"),
		DTStart:               ev.StartDateTime,
		DTEnd:                 ev.EndDateTime,
		IsAllDay:              ev.IsAllDayEvent,
		Source:                source,
		CalendarEventID:       ev.CalendarEventId,
		QuizID:                quizID,
		IsRecurring:           ev.IsRecurring,
		RepeatType: func() int {
			if ev.RecurrenceInfo != nil {
				return ev.RecurrenceInfo.RepeatType
			}
			return 1
		}(),
		RepeatEvery: func() int {
			if ev.RecurrenceInfo != nil {
				return ev.RecurrenceInfo.RepeatEvery
			}
			return 0
		}(),
		RepeatOnInfo:          ev.RecurrenceInfo,
		RepeatUntilDateString: func() string {
			if ev.RecurrenceInfo != nil {
				return ev.RecurrenceInfo.RepeatUntilDate
			}
			return ""
		}(),
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



// QuizToStringEntry converts a QuizAPI to a BrightSpaceStringEntry for event conversion.
func QuizToStringEntry(quiz QuizAPI) BrightSpaceStringEntry {
	descParts := []string{}
	if quiz.Description.Text.Text != "" {
		plainDesc := htmlToPlainText(quiz.Description.Text.Html)
		if plainDesc != "" {
			descParts = append(descParts, plainDesc)
		}
	}

	var infoParts []string
	if quiz.SubmissionTimeLimit.IsEnforced && quiz.SubmissionTimeLimit.TimeLimitValue > 0 {
		infoParts = append(infoParts, fmt.Sprintf("Duration: %d minutes", quiz.SubmissionTimeLimit.TimeLimitValue))
	}
	if quiz.AttemptsAllowed.IsUnlimited {
		infoParts = append(infoParts, "Unlimited attempts")
	} else if quiz.AttemptsAllowed.NumberOfAttemptsAllowed > 0 {
		infoParts = append(infoParts, fmt.Sprintf("%d attempts allowed", quiz.AttemptsAllowed.NumberOfAttemptsAllowed))
	}

	if len(infoParts) > 0 {
		descParts = append(descParts, strings.Join(infoParts, ", "))
	}

	dtEnd := quiz.DueDate
	if quiz.EndDate != "" {
		dtEnd = quiz.EndDate
	}

	return BrightSpaceStringEntry{
		Title:           quiz.Name,
		OrgUnitId:       quiz.OrgUnitId,
		OrgUnitName:     quiz.OrgUnitName,
		OrgUnitCode:     quiz.OrgUnitCode,
		Location:        "",
		Description:     strings.Join(descParts, "\n"),
		DTStart:         quiz.StartDate,
		DTEnd:           dtEnd,
		IsAllDay:        false,
		Source:          "brightspace-quizzes",
		CalendarEventID: 0,
		QuizID:          quiz.QuizId,
	}
}

// htmlToPlainText strips HTML tags and common entities from an HTML string.
func htmlToPlainText(html string) string {
	// Convert line breaks
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br />", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n")
	html = strings.ReplaceAll(html, "<p>", "")

	// Strip HTML tags
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

	// Handle common HTML entities (order matters: named before numeric)
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&ensp;", " ")
	text = strings.ReplaceAll(text, "&emsp;", "  ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&apos;", "'")
	text = strings.ReplaceAll(text, "&ndash;", "-")
	text = strings.ReplaceAll(text, "&mdash;", "-")
	text = strings.ReplaceAll(text, "&hellip;", "...")
	text = strings.ReplaceAll(text, "&bull;", "-")
	text = strings.ReplaceAll(text, "&copy;", "(C)")
	text = strings.ReplaceAll(text, "&reg;", "(R)")
	text = strings.ReplaceAll(text, "&trade;", "(TM)")
	text = strings.ReplaceAll(text, "&minus;", "-")
	text = strings.ReplaceAll(text, "&times;", "x")
	text = strings.ReplaceAll(text, "&divide;", "/")

	// Convert &#160; to regular space before stripNumericEntities
	// (otherwise it becomes U+00A0 which EscapeText strips as non-ASCII)
	text = strings.ReplaceAll(text, "&#160;", " ")

	// Handle numeric character references: &#NNN; and &#xHHH;
	text = stripNumericEntities(text)

	// Strip any remaining HTML entities (&...;)
	text = stripRemainingEntities(text)

	return strings.TrimSpace(text)
}

// stripNumericEntities converts numeric character references (&#NNN; and &#xHHH;)
// to their Unicode equivalents, then returns the string.
func stripNumericEntities(s string) string {
	// Handle decimal: &#NNN;
	for {
		idx := strings.Index(s, "&#")
		if idx < 0 {
			break
		}
		semiIdx := strings.IndexByte(s[idx:], ';')
		if semiIdx < 0 {
			break
		}
		numStr := s[idx+2 : idx+1+semiIdx]
		var r rune
		if len(numStr) > 0 && (numStr[0] == 'x' || numStr[0] == 'X') {
			// Hex: &#xHHH;
			var n uint64
			fmt.Sscanf(numStr[1:], "%x", &n)
			r = rune(n)
		} else {
			// Decimal: &#NNN;
			var n uint64
			fmt.Sscanf(numStr, "%d", &n)
			r = rune(n)
		}
		s = s[:idx] + string(r) + s[idx+1+semiIdx:]
	}
	return s
}

// stripRemainingEntities removes any remaining HTML entity patterns.
func stripRemainingEntities(s string) string {
	result := strings.Builder{}
	i := 0
	for i < len(s) {
		if s[i] == '&' {
			semiIdx := strings.IndexByte(s[i:], ';')
			if semiIdx > 0 && semiIdx < 10 {
				// Skip the entity
				i += 1 + semiIdx
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// BrightSpaceStringEntry represents a BrightSpace event with string-based timestamps,
// as returned by the browser scraping layer.
type BrightSpaceStringEntry struct {
	Title                 string
	OrgUnitId             string
	OrgUnitName           string
	OrgUnitCode           string
	Location              string
	Description           string
	DTStart               string
	DTEnd                 string
	IsAllDay              bool
	Source                string
	CalendarEventID       int
	QuizID                int
	IsRecurring           bool
	RepeatType            int
	RepeatEvery           int
	RepeatOnInfo          *RecurrenceInfo
	RepeatUntilDateString string
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

// parseEntryTimes parses DTStart and DTEnd from a BrightSpaceStringEntry.
// Returns (dtStart, dtEnd, error). If DTStart is empty but DTEnd exists, DTEnd is used as DTStart.
func parseEntryTimes(entry *BrightSpaceStringEntry, loc *time.Location) (time.Time, time.Time, error) {
	if entry.DTStart == "" && entry.DTEnd != "" {
		entry.DTStart = entry.DTEnd
	}

	dtStart, err := ParseTimestamp(entry.DTStart, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
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

	// Zero-duration BrightSpace events: set DTSTART to 1 hour before DTEND.
	if dtStart.Equal(dtEnd) && !entry.IsAllDay {
		dtStart = dtEnd.Add(-1 * time.Hour)
	}

	return dtStart, dtEnd, nil
}

// buildEvent creates a calendar.Event from parsed times and a BrightSpaceStringEntry.
func buildEvent(entry BrightSpaceStringEntry, dtStart, dtEnd time.Time, recurrenceIndex int) calendar.Event {
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

	return calendar.Event{
		CourseCode:      entry.OrgUnitCode,
		DTStart:         dtStart,
		DTEnd:           dtEnd,
		Summary:         summary,
		Title:           entry.Title,
		OrgUnitID:       entry.OrgUnitId,
		OrgUnitName:     entry.OrgUnitName,
		OrgUnitCode:     entry.OrgUnitCode,
		Location:        entry.Location,
		Description:     entry.Description,
		Source:          entry.Source,
		CalendarEventID: entry.CalendarEventID,
		QuizID:          entry.QuizID,
		RecurrenceIndex: recurrenceIndex,
	}
}

// singleEventFromEntry creates a calendar.Event from a BrightSpaceStringEntry using
// the entry's original DTStart/DTEnd (parsed). Returns the event and an error if parsing fails.
func singleEventFromEntry(entry BrightSpaceStringEntry, loc *time.Location) (calendar.Event, error) {
	dtStart, dtEnd, err := parseEntryTimes(&entry, loc)
	if err != nil {
		return calendar.Event{}, err
	}
	return buildEvent(entry, dtStart, dtEnd, 0), nil
}

// EntriesToEvents converts BrightSpace string entries to calendar.Event values,
// applying blocklist filtering and expanding recurring events.
func EntriesToEvents(entries []BrightSpaceStringEntry, blocklist *Blocklist, loc *time.Location) []calendar.Event {
	events := make([]calendar.Event, 0, len(entries))
	for _, entry := range entries {
		if blocklist.IsCourseBlocked(entry.OrgUnitId, entry.OrgUnitName, entry.OrgUnitCode) {
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
		if blocklist.IsQuizBlocked(entry.Title) {
			log.Printf("brightspace: blocked quiz %q in %s", entry.Title, entry.OrgUnitName)
			continue
		}

		dtStart, dtEnd, err := parseEntryTimes(&entry, loc)
		if err != nil {
			log.Printf("brightspace: skipping entry %q: invalid start time %q: %v", entry.Title, entry.DTStart, err)
			continue
		}

		var entryEvents []calendar.Event
		if entry.IsRecurring && entry.RepeatType > 0 {
			expanded, err := ExpandRecurrence(entry, loc)
			if err != nil {
				log.Printf("brightspace: expanding recurring event %q failed: %v", entry.Title, err)
				entryEvents = []calendar.Event{buildEvent(entry, dtStart, dtEnd, 0)}
			} else {
				entryEvents = expanded
			}
		} else {
			entryEvents = []calendar.Event{buildEvent(entry, dtStart, dtEnd, 0)}
		}
		events = append(events, entryEvents...)
	}
	return events
}

// QuizEntriesToEvents converts BrightSpace quiz string entries to calendar.Event values,
// applying blocklist filtering. This is an alias for EntriesToEvents that makes
// the quiz source explicit at the call site.
func QuizEntriesToEvents(entries []BrightSpaceStringEntry, blocklist *Blocklist, loc *time.Location) []calendar.Event {
	return EntriesToEvents(entries, blocklist, loc)
}
