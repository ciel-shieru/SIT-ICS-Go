package ics

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type ICSCache struct {
	mu   sync.RWMutex
	data []byte
	dirty bool
}

func NewICSCache() *ICSCache {
	return &ICSCache{}
}

func (c *ICSCache) LoadFromFile(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read ICS file: %w", err)
	}

	c.data = data
	c.dirty = false
	return nil
}

func (c *ICSCache) Get() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

func (c *ICSCache) Update(events []Event, tz string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	merged, err := c.merge(events)
	if err != nil {
		return fmt.Errorf("merge events: %w", err)
	}

	newData, err := Write(merged, tz)
	if err != nil {
		return fmt.Errorf("write ICS: %w", err)
	}

	c.data = newData
	c.dirty = true
	return nil
}

func (c *ICSCache) merge(newEvents []Event) ([]Event, error) {
	existing := c.parseExisting()

	merged := make([]Event, 0, len(existing)+len(newEvents))
	newUIDs := make(map[string]bool)

	for _, event := range newEvents {
		event.UID = GenerateUID(event)
		merged = append(merged, event)
		newUIDs[event.UID] = true
	}

	for uid, event := range existing {
		if !newUIDs[uid] {
			merged = append(merged, event)
		}
	}

	return merged, nil
}

func (c *ICSCache) parseExisting() map[string]Event {
	existing := make(map[string]Event)

	if len(c.data) == 0 {
		return existing
	}

	content := string(c.data)
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
			existing[event.UID] = event
		}
	}

	return existing
}

func (c *ICSCache) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, c.data, 0600); err != nil {
		return fmt.Errorf("write temp ICS file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename ICS file: %w", err)
	}

	c.dirty = false
	return nil
}

func unescapeText(text string) string {
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\,", ",")
	text = strings.ReplaceAll(text, "\\;", ";")
	text = strings.ReplaceAll(text, "\\\\", "\\")
	return text
}
