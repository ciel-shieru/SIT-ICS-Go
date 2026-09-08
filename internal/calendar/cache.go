package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"time"
)

type ICSCache struct {
	mu     sync.RWMutex
	events []Event
	dirty  bool
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

	c.events = parseICS(data)
	c.dirty = false
	return nil
}

func (c *ICSCache) Get(tz string, refreshInterval time.Duration) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, _ := Write(c.events, tz, refreshInterval)
	return data
}

func (c *ICSCache) GetFiltered(tz string, filterFn func(Event) bool, refreshInterval time.Duration) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	filtered := filterEvents(c.events, filterFn)
	data, _ := Write(filtered, tz, refreshInterval)
	return data
}

func (c *ICSCache) EventCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

func (c *ICSCache) DeleteByPredicate(filterFn func(Event) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	deleted := 0
	filtered := make([]Event, 0, len(c.events))
	for _, event := range c.events {
		if filterFn(event) {
			deleted++
		} else {
			filtered = append(filtered, event)
		}
	}

	c.events = filtered
	c.dirty = true
	return deleted
}

func (c *ICSCache) Update(events []Event, tz string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	merged, err := c.merge(events)
	if err != nil {
		return fmt.Errorf("merge events: %w", err)
	}

	c.events = merged
	c.dirty = true
	return nil
}

func (c *ICSCache) SaveToFile(path string, refreshInterval time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	data, _ := Write(c.events, "UTC", refreshInterval)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write temp ICS file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename ICS file: %w", err)
	}

	c.dirty = false
	return nil
}

func (c *ICSCache) SaveAllToFiles(mainPath, onlinePath, campusPath string, tz string, refreshInterval time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	mainData, _ := Write(c.events, tz, refreshInterval)
	onlineData, _ := Write(filterEvents(c.events, IsOnline), tz, refreshInterval)
	campusData, _ := Write(filterEvents(c.events, IsNotOnline), tz, refreshInterval)

	for path, data := range map[string][]byte{
		mainPath:   mainData,
		onlinePath: onlineData,
		campusPath: campusData,
	} {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0600); err != nil {
			return fmt.Errorf("write temp ICS file %q: %w", tmp, err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("rename ICS file %q: %w", path, err)
		}
	}

	c.dirty = false
	return nil
}

func (c *ICSCache) SaveAllWithXsiteFiles(mainPath, onlinePath, campusPath, xsiteEventsPath, xsiteDropboxPath string, tz string, refreshInterval time.Duration) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.dirty {
		return nil
	}

	mainData, _ := Write(c.events, tz, refreshInterval)
	onlineData, _ := Write(filterEvents(c.events, IsOnline), tz, refreshInterval)
	campusData, _ := Write(filterEvents(c.events, IsNotOnlineAndNotBrightSpace), tz, refreshInterval)
	xsiteEventsData, _ := Write(filterEventsBySource(c.events, "brightspace-calendar"), tz, refreshInterval)
	xsiteDropboxData, _ := Write(filterEventsBySource(c.events, "brightspace-dropbox"), tz, refreshInterval)

	for path, data := range map[string][]byte{
		mainPath:            mainData,
		onlinePath:          onlineData,
		campusPath:          campusData,
		xsiteEventsPath:     xsiteEventsData,
		xsiteDropboxPath:    xsiteDropboxData,
	} {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0600); err != nil {
			return fmt.Errorf("write temp ICS file %q: %w", tmp, err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("rename ICS file %q: %w", path, err)
		}
	}

	c.dirty = false
	return nil
}

func generateUID(event Event) string {
	var uidStr string
	if event.Source != "" {
		uidStr = fmt.Sprintf("%s-%s-%s-%s-%s-%s",
			event.Source,
			event.Summary,
			event.Location,
			event.DTStart.Format("2006-01-02"),
			event.DTStart.Format("15:04"),
			event.DTEnd.Format("15:04"),
		)
	} else {
		uidStr = fmt.Sprintf("%s-%s-%s-%s-%s",
			event.Summary,
			event.Location,
			event.DTStart.Format("2006-01-02"),
			event.DTStart.Format("15:04"),
			event.DTEnd.Format("15:04"),
		)
	}
	hash := sha256.Sum256([]byte(uidStr))
	return hex.EncodeToString(hash[:])
}

func (c *ICSCache) merge(newEvents []Event) ([]Event, error) {
	existing := c.parseExisting()

	merged := make([]Event, 0, len(existing)+len(newEvents))
	newUIDs := make(map[string]bool)

	for _, event := range newEvents {
		event.UID = generateUID(event)
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
	for _, event := range c.events {
		existing[event.UID] = event
	}
	return existing
}
