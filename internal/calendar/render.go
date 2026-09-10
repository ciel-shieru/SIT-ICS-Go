package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type RenderOptions struct {
	Timezone        *time.Location
	RefreshInterval time.Duration
}

func EventID(e Event) string {
	var uidStr string
	if e.Source != "" {
		uidStr = fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			e.Source,
			e.Summary,
			e.Location,
			e.DTStart.Format("2006-01-02"),
			e.DTStart.Format("15:04"),
			e.DTEnd.Format("15:04"),
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
	sb.WriteString(fmt.Sprintf("REFRESH-INTERVAL;VALUE=DURATION:%s\r\n", formatDuration(opts.RefreshInterval)))

	for _, event := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		sb.WriteString(fmt.Sprintf("DTSTART;TZID=%s:%s\r\n", opts.Timezone.String(), toICSTime(event.DTStart, opts.Timezone)))
		sb.WriteString(fmt.Sprintf("DTEND;TZID=%s:%s\r\n", opts.Timezone.String(), toICSTime(event.DTEnd, opts.Timezone)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", EscapeText(event.Summary)))
		if event.Location != "" {
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", EscapeText(event.Location)))
		}
		if event.Description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", EscapeText(event.Description)))
		}
		if event.Source != "" {
			sb.WriteString(fmt.Sprintf("X-SOURCE:%s\r\n", EscapeText(event.Source)))
		}
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return []byte(sb.String()), nil
}

func toICSTime(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("20060102T150405")
}

// EscapeText escapes ICS special characters in text per RFC 5545.
func EscapeText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
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
