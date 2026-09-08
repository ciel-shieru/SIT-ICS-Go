package calendar

import (
	"os"
	"strings"
	"time"
)

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func parseICS(data []byte) []Event {
	events := make([]Event, 0)
	content := string(data)
	eventBlocks := strings.Split(content, "BEGIN:VEVENT")

	for _, block := range eventBlocks {
		if !strings.Contains(block, "END:VEVENT") {
			continue
		}

		event := Event{}
		lines := strings.Split(block, "\r\n")

		for _, line := range lines {
			line = strings.TrimPrefix(line, "END:VEVENT")
			line = strings.TrimPrefix(line, "BEGIN:VEVENT")

			if strings.HasPrefix(line, "UID:") {
				event.UID = strings.TrimPrefix(line, "UID:")
			} else if strings.HasPrefix(line, "SUMMARY:") {
				event.Summary = unescapeText(strings.TrimPrefix(line, "SUMMARY:"))
			} else if strings.HasPrefix(line, "LOCATION:") {
				event.Location = unescapeText(strings.TrimPrefix(line, "LOCATION:"))
			} else if strings.HasPrefix(line, "DESCRIPTION:") {
				event.Description = unescapeText(strings.TrimPrefix(line, "DESCRIPTION:"))
			} else if strings.HasPrefix(line, "X-SOURCE:") {
				event.Source = unescapeText(strings.TrimPrefix(line, "X-SOURCE:"))
			} else if strings.HasPrefix(line, "DTSTART;TZID=") {
				timeStr := strings.SplitN(line, ":", 2)
				if len(timeStr) == 2 {
					t, err := time.ParseInLocation("20060102T150405", timeStr[1], time.Local)
					if err == nil {
						event.DTStart = t
					}
				}
			} else if strings.HasPrefix(line, "DTEND;TZID=") {
				timeStr := strings.SplitN(line, ":", 2)
				if len(timeStr) == 2 {
					t, err := time.ParseInLocation("20060102T150405", timeStr[1], time.Local)
					if err == nil {
						event.DTEnd = t
					}
				}
			}
		}

		if event.UID != "" {
			events = append(events, event)
		}
	}

	return events
}

func unescapeText(text string) string {
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\,", ",")
	text = strings.ReplaceAll(text, "\\;", ";")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}
