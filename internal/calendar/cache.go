package calendar

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

type ICSCache struct {
	mu     sync.RWMutex
	events []Event
	dirty  bool
}

type Outputs struct {
	Main       string
	Online     string
	Campus     string
	BSEvents   string
	BSDropbox  string
}

type outputErrors []outputError

type outputError struct {
	path string
	err  error
}

func (e outputErrors) Error() string {
	msgs := make([]string, 0, len(e))
	for _, o := range e {
		msgs = append(msgs, fmt.Sprintf("%s: %v", o.path, o.err))
	}
	return fmt.Sprintf("output errors: %v", msgs)
}

func NewICSCache() *ICSCache {
	return &ICSCache{}
}

// sortEvents sorts events deterministically in-place:
// primary by DTStart (ascending), secondary by Location (ascending),
// tiebreaker by UID (ascending).
func sortEvents(events []Event) {
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].DTStart.Equal(events[j].DTStart) {
			return events[i].DTStart.Before(events[j].DTStart)
		}
		if events[i].Location != events[j].Location {
			return events[i].Location < events[j].Location
		}
		return events[i].UID < events[j].UID
	})
}

func (c *ICSCache) LoadFromFile(path string, loc *time.Location) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read ICS file: %w", err)
	}

	c.events = parseICS(data, loc)
	c.dirty = false
	return nil
}

func (c *ICSCache) Get(tz string, refreshInterval time.Duration) []byte {
	c.mu.RLock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	sortEvents(events)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
	}
	data, err := Render(events, RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return nil
	}
	return data
}

func (c *ICSCache) GetFiltered(tz string, filterFn func(Event) bool, refreshInterval time.Duration) []byte {
	c.mu.RLock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	filtered := filterEvents(events, filterFn)
	sortEvents(filtered)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
	}
	data, err := Render(filtered, RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return nil
	}
	return data
}

func (c *ICSCache) EventCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// RemoveWhere removes events matching the predicate and returns the count removed.
// This is an explicit deletion operation, distinct from merge retention semantics.
func (c *ICSCache) RemoveWhere(predicate func(Event) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	deleted := 0
	filtered := make([]Event, 0, len(c.events))
	for _, event := range c.events {
		if predicate(event) {
			deleted++
		} else {
			filtered = append(filtered, event)
		}
	}

	c.events = filtered
	c.dirty = true
	return deleted
}

// Update merges new events into the cache. The merge is non-destructive:
// events absent from the latest upstream response are intentionally retained.
// Matching events (by UID) are updated; new events are added; duplicates are avoided.
func (c *ICSCache) Update(events []Event, tz string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	merged, err := c.mergeEvents(events)
	if err != nil {
		return fmt.Errorf("merge events: %w", err)
	}

	c.events = merged
	c.dirty = true
	return nil
}

// SaveToFile atomically writes the ICS file if the cache is dirty.
// The dirty flag is reset under an exclusive lock after writing.
func (c *ICSCache) SaveToFile(path string, refreshInterval time.Duration) error {
	c.mu.RLock()
	dirty := c.dirty
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	if !dirty {
		return nil
	}

	utc := time.UTC
	data, err := Render(events, RenderOptions{
		Timezone:        utc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render ICS: %w", err)
	}

	if err := writeAtomic(path, data); err != nil {
		return fmt.Errorf("write ICS file %q: %w", path, err)
	}

	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()
	return nil
}

// SaveAllToFiles atomically writes the main, online, and campus ICS files if dirty.
// The dirty flag is reset under an exclusive lock after writing.
func (c *ICSCache) SaveAllToFiles(mainPath, onlinePath, campusPath string, tz string, refreshInterval time.Duration) error {
	c.mu.RLock()
	dirty := c.dirty
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	if !dirty {
		return nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
	}

	mainData, err := Render(events, RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render main ICS: %w", err)
	}
	onlineData, err := Render(filterEvents(events, IsOnline), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render online ICS: %w", err)
	}
	campusData, err := Render(filterEvents(events, IsNotOnline), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render campus ICS: %w", err)
	}

	for path, data := range map[string][]byte{
		mainPath:   mainData,
		onlinePath: onlineData,
		campusPath: campusData,
	} {
		if err := writeAtomic(path, data); err != nil {
			return fmt.Errorf("write ICS file %q: %w", path, err)
		}
	}

	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()
	return nil
}

// SaveOutputs atomically writes all configured ICS output files if dirty.
// Uses writeAtomic for individual file persistence.
// If any output fails, the cache dirty flag is preserved and all errors are returned.
func (c *ICSCache) SaveOutputs(out Outputs, tz string, refreshInterval time.Duration) error {
	c.mu.RLock()
	dirty := c.dirty
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	if !dirty {
		return nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
	}

	mainData, err := Render(events, RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render main ICS: %w", err)
	}
	onlineData, err := Render(filterEvents(events, IsOnline), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render online ICS: %w", err)
	}
	campusData, err := Render(filterEvents(events, IsNotOnlineAndNotBrightSpace), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render campus ICS: %w", err)
	}
	bsEventsData, err := Render(filterEventsBySource(events, "brightspace-calendar"), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render brightspace-events ICS: %w", err)
	}
	bsDropboxData, err := Render(filterEventsBySource(events, "brightspace-dropbox"), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render brightspace-dropbox ICS: %w", err)
	}

	type fileOutput struct {
		path string
		data []byte
	}

	files := []fileOutput{
		{out.Main, mainData},
		{out.Online, onlineData},
		{out.Campus, campusData},
		{out.BSEvents, bsEventsData},
		{out.BSDropbox, bsDropboxData},
	}

	var errs outputErrors
	for _, f := range files {
		if f.path == "" {
			continue
		}
		if err := writeAtomic(f.path, f.data); err != nil {
			errs = append(errs, outputError{path: f.path, err: err})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()
	return nil
}

// mergeEvents merges newEvents into existing cache events with non-destructive semantics:
// - Events in newEvents are added or update matching existing events (by UID).
// - Events absent from newEvents are retained (non-destructive).
// - No duplicates are created.
// This does NOT implement ReplaceSource or authoritative source reconciliation.
func (c *ICSCache) mergeEvents(newEvents []Event) ([]Event, error) {
	existing := c.indexByUID()

	merged := make([]Event, 0, len(existing)+len(newEvents))
	seenUIDs := make(map[string]bool)

	for _, event := range newEvents {
		event.UID = EventID(event)
		merged = append(merged, event)
		seenUIDs[event.UID] = true
	}

	for uid, event := range existing {
		if !seenUIDs[uid] {
			merged = append(merged, event)
		}
	}

	return merged, nil
}

// indexByUID builds a map from UID to Event for all events currently in the cache.
func (c *ICSCache) indexByUID() map[string]Event {
	index := make(map[string]Event)
	for _, event := range c.events {
		index[event.UID] = event
	}
	return index
}
