package calendar

import (
	"fmt"
	"strings"
	"time"
)

func Write(events []Event, tz string, refreshInterval time.Duration) ([]byte, error) {
	var sb strings.Builder

	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//SIT Timetable//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")
	sb.WriteString(fmt.Sprintf("REFRESH-INTERVAL;VALUE=DURATION:%s\r\n", formatDuration(refreshInterval)))

	for _, event := range events {
		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		sb.WriteString(fmt.Sprintf("DTSTART;TZID=%s:%s\r\n", tz, toICSTime(event.DTStart)))
		sb.WriteString(fmt.Sprintf("DTEND;TZID=%s:%s\r\n", tz, toICSTime(event.DTEnd)))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeText(event.Summary)))
		if event.Location != "" {
			sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeText(event.Location)))
		}
		if event.Description != "" {
			sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeText(event.Description)))
		}
		if event.Source != "" {
			sb.WriteString(fmt.Sprintf("X-SOURCE:%s\r\n", escapeText(event.Source)))
		}
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	return []byte(sb.String()), nil
}

func toICSTime(t time.Time) string {
	return t.Format("20060102T150405")
}

func escapeText(text string) string {
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
