package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type AlertAction int

const (
	AlertDisplay AlertAction = iota
)

type RenderOptions struct {
	Timezone        *time.Location
	RefreshInterval time.Duration
	Alerts          []Alert
}

type Alert struct {
	Duration    time.Duration
	Action      AlertAction
	Description string
}

func EventID(e Event) string {
	var uidStr string
	if e.Source != "" {
		if e.RecurrenceIndex > 0 {
			uidStr = fmt.Sprintf("%s-%s-%s-%s-%s-%d",
				e.Source,
				e.Summary,
				e.Location,
				e.DTStart.Format("2006-01-02"),
				e.DTStart.Format("15:04"),
				e.RecurrenceIndex,
			)
		} else {
			uidStr = fmt.Sprintf("%s-%s-%s-%s-%s-%s",
				e.Source,
				e.Summary,
				e.Location,
				e.DTStart.Format("2006-01-02"),
				e.DTStart.Format("15:04"),
				e.DTEnd.Format("15:04"),
			)
		}
	} else {
		if e.RecurrenceIndex > 0 {
			uidStr = fmt.Sprintf("%s-%s-%s-%s-%d",
				e.Summary,
				e.Location,
				e.DTStart.Format("2006-01-02"),
				e.DTStart.Format("15:04"),
				e.RecurrenceIndex,
			)
		} else {
			uidStr = fmt.Sprintf("%s-%s-%s-%s-%s",
				e.Summary,
				e.Location,
				e.DTStart.Format("2006-01-02"),
				e.DTStart.Format("15:04"),
				e.DTEnd.Format("15:04"),
			)
		}
	}
	hash := sha256.Sum256([]byte(uidStr))
	return hex.EncodeToString(hash[:])
}

func Render(events []Event, opts RenderOptions) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//SIT Timetable//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	if vtz := vtimezoneBlock(opts.Timezone); vtz != "" {
		sb.WriteString(vtz)
	}
	sb.WriteString(fmt.Sprintf("REFRESH-INTERVAL;VALUE=DURATION:%s\r\n", formatDuration(opts.RefreshInterval)))

	for _, event := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", event.DTStamp.UTC().Format("20060102T150405Z")))
		sb.WriteString(fmt.Sprintf("DTSTART;TZID=%s:%s\r\n", opts.Timezone.String(), toICSTime(event.DTStart, opts.Timezone)))
		sb.WriteString(fmt.Sprintf("DTEND;TZID=%s:%s\r\n", opts.Timezone.String(), toICSTime(event.DTEnd, opts.Timezone)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", FoldText(EscapeText(event.Summary))))
		if event.Location != "" {
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", FoldText(EscapeText(event.Location))))
		}
		if event.Description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", FoldText(EscapeText(event.Description))))
		}
		if event.Source != "" {
			sb.WriteString(fmt.Sprintf("X-SOURCE:%s\r\n", FoldText(EscapeText(event.Source))))
		}
		if event.OrgUnitID != "" {
			sb.WriteString(fmt.Sprintf("X-OrgUnitID:%s\r\n", FoldText(EscapeText(event.OrgUnitID))))
		}
		if event.OrgUnitName != "" {
			sb.WriteString(fmt.Sprintf("X-OrgUnitName:%s\r\n", FoldText(EscapeText(event.OrgUnitName))))
		}
		if event.OrgUnitCode != "" {
			sb.WriteString(fmt.Sprintf("X-OrgUnitCode:%s\r\n", FoldText(EscapeText(event.OrgUnitCode))))
		}
		if event.Title != "" {
			sb.WriteString(fmt.Sprintf("X-Title:%s\r\n", FoldText(EscapeText(event.Title))))
		}
		if event.CalendarEventID > 0 {
			sb.WriteString(fmt.Sprintf("X-CalendarEventId:%d\r\n", event.CalendarEventID))
		}
		if event.QuizID > 0 {
			sb.WriteString(fmt.Sprintf("X-QuizId:%d\r\n", event.QuizID))
		}
		for _, alert := range opts.Alerts {
			renderVALARM(&sb, alert)
		}
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return []byte(sb.String()), nil
}

func toICSTime(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("20060102T150405")
}

func vtimezoneBlock(loc *time.Location) string {
	if loc == time.UTC || loc.String() == "UTC" {
		return ""
	}
	return "BEGIN:VTIMEZONE\r\nTZID:Asia/Singapore\r\nBEGIN:STANDARD\r\nDTSTART:19820101T000000\r\nTZOFFSETFROM:+0800\r\nTZOFFSETTO:+0800\r\nTZNAME:SGT\r\nEND:STANDARD\r\nEND:VTIMEZONE\r\n"
}

// EscapeText escapes ICS special characters in text per RFC 5545.
func EscapeText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

// FoldText folds a content line to comply with RFC 5545 Section 3.1
// (max 75 octets per line). Each continuation line is prefixed with CRLF+space.
func FoldText(text string) string {
	if len(text) <= 75 {
		return text
	}
	var result strings.Builder
	for len(text) > 0 {
		if result.Len() == 0 {
			if len(text) > 75 {
				result.WriteString(text[:75])
				text = text[75:]
			} else {
				result.WriteString(text)
				text = ""
			}
		} else {
			if len(text) > 74 {
				result.WriteString("\r\n " + text[:74])
				text = text[74:]
			} else {
				result.WriteString("\r\n " + text)
				text = ""
			}
		}
	}
	return result.String()
}

func formatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("PT%dH%dM", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("PT%dH", hours)
	}
	return fmt.Sprintf("PT%dM", minutes)
}

func renderVALARM(sb *strings.Builder, alert Alert) {
	sb.WriteString("BEGIN:VALARM\r\n")
	sb.WriteString("ACTION:DISPLAY\r\n")
	sb.WriteString(fmt.Sprintf("TRIGGER:%s\r\n", formatICSDuration(alert.Duration)))
	if alert.Description != "" {
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", FoldText(EscapeText(alert.Description))))
	}
	sb.WriteString("END:VALARM\r\n")
}

func formatICSDuration(d time.Duration) string {
	neg := d < 0
	if neg {
		d = -d
	}
	totalMinutes := int(d.Minutes())
	days := totalMinutes / (24 * 60)
	remaining := totalMinutes % (24 * 60)
	hours := remaining / 60
	minutes := remaining % 60

	var prefix string
	if neg {
		prefix = "-"
	}

	if days > 0 {
		if hours > 0 || minutes > 0 {
			return fmt.Sprintf("%sP%dDT%dH%dM", prefix, days, hours, minutes)
		}
		return fmt.Sprintf("%sP%dD", prefix, days)
	}
	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%sPT%dH%dM", prefix, hours, minutes)
		}
		return fmt.Sprintf("%sPT%dH", prefix, hours)
	}
	return fmt.Sprintf("%sPT%dM", prefix, minutes)
}
