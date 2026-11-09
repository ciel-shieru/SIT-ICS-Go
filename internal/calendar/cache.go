package calendar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type ICSCache struct {
	mu           sync.RWMutex
	events       []Event
	dirty        bool
	lastModified time.Time
}

type Outputs struct {
	Main       string
	Online     string
	Campus     string
	XsiteEvents string
	XsiteDropbox string
	XsiteQuizzes string
	Xsite      string
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

func ComputeETag(data []byte) string {
	hash := sha256.Sum256(data)
	return `W/"` + hex.EncodeToString(hash[:]) + `"`
}

func (c *ICSCache) GetLastModified() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastModified
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

	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat ICS file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read ICS file: %w", err)
	}

	c.events = parseICS(data, loc)
	if fi != nil {
		c.lastModified = fi.ModTime()
	}
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
		loc = time.UTC
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
		loc = time.UTC
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

func (c *ICSCache) GetWithAlerts(tz string, refreshInterval time.Duration, alerts []Alert) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Singapore")
	}
	return Render(events, RenderOptions{Timezone: loc, RefreshInterval: refreshInterval, Alerts: alerts})
}

func (c *ICSCache) GetFilteredWithAlerts(filterFn func(Event) bool, tz string, refreshInterval time.Duration, alerts []Alert) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	filtered := filterEvents(events, filterFn)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Singapore")
	}
	return Render(filtered, RenderOptions{Timezone: loc, RefreshInterval: refreshInterval, Alerts: alerts})
}

func (c *ICSCache) EventCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.events)
}

// GetEvents returns a deterministic copy of all cached events, sorted by
// DTStart, then Location, then UID. The returned slice is safe to modify.
func (c *ICSCache) GetEvents() []Event {
	c.mu.RLock()
	events := make([]Event, len(c.events))
	copy(events, c.events)
	c.mu.RUnlock()

	sortEvents(events)
	return events
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

// Update merges new events into the cache without deleting events absent from
// the latest response. Use UpdateWithSourceReconciliation when the caller has
// completed an authoritative upstream fetch and can identify the covered sources.
func (c *ICSCache) Update(events []Event, _ string) error {
	return c.UpdateWithSourceReconciliation(events, nil, time.Now())
}

// UpdateWithSourceReconciliation merges new events and reconciles only the
// sources listed in authoritativeSources. Missing events from those sources
// are removed only when their DTSTART is after now; events whose start time has
// passed (including events currently in progress) are retained. Sources not listed are untouched,
// which protects cached events when a source is disabled or its fetch is partial
// or unsuccessful.
func (c *ICSCache) UpdateWithSourceReconciliation(events []Event, authoritativeSources []string, now time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	sourceSet := make(map[string]struct{}, len(authoritativeSources))
	for _, source := range authoritativeSources {
		sourceSet[source] = struct{}{}
	}

	merged, err := c.mergeEvents(events, sourceSet, now)
	if err != nil {
		return fmt.Errorf("merge events: %w", err)
	}

	c.events = merged
	c.lastModified = time.Now().UTC()
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
func (c *ICSCache) SaveOutputs(out Outputs, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts []Alert) error {
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
		Alerts:          mainAlerts,
	})
	if err != nil {
		return fmt.Errorf("render main ICS: %w", err)
	}
	onlineData, err := Render(filterEvents(events, IsOnline), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
		Alerts:          onlineAlerts,
	})
	if err != nil {
		return fmt.Errorf("render online ICS: %w", err)
	}
	campusData, err := Render(filterEvents(events, IsCampus), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
		Alerts:          campusAlerts,
	})
	if err != nil {
		return fmt.Errorf("render campus ICS: %w", err)
	}
	xsiteEventsData, err := Render(filterEventsBySource(events, "brightspace-calendar"), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
		Alerts:          xsiteEventsAlerts,
	})
	if err != nil {
		return fmt.Errorf("render brightspace-events ICS: %w", err)
	}
	xsiteDropboxData, err := Render(filterEventsBySource(events, "brightspace-dropbox"), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
		Alerts:          xsiteDropboxAlerts,
	})
	if err != nil {
		return fmt.Errorf("render brightspace-dropbox ICS: %w", err)
	}
	xsiteQuizzesData, err := Render(filterEventsBySource(events, "brightspace-quizzes"), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
		Alerts:          xsiteQuizzesAlerts,
	})
	if err != nil {
		return fmt.Errorf("render brightspace-quizzes ICS: %w", err)
	}

	type fileOutput struct {
		path string
		data []byte
	}

	xsiteComboData, err := Render(filterEvents(events, func(e Event) bool { return strings.HasPrefix(e.Source, "brightspace-") }), RenderOptions{
		Timezone:        loc,
		RefreshInterval: refreshInterval,
	})
	if err != nil {
		return fmt.Errorf("render xsite combo ICS: %w", err)
	}

	files := []fileOutput{
		{out.Main, mainData},
		{out.Online, onlineData},
		{out.Campus, campusData},
		{out.XsiteEvents, xsiteEventsData},
		{out.XsiteDropbox, xsiteDropboxData},
		{out.XsiteQuizzes, xsiteQuizzesData},
		{out.Xsite, xsiteComboData},
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

// mergeEvents merges newEvents into the existing cache by UID. Missing events
// from authoritative sources are removed only when their DTSTART is upcoming;
// historical events and events from non-authoritative sources are retained.
func (c *ICSCache) mergeEvents(newEvents []Event, authoritativeSources map[string]struct{}, now time.Time) ([]Event, error) {
	existing := c.indexByUID()

	merged := make([]Event, 0, len(existing)+len(newEvents))
	seenUIDs := make(map[string]bool)

	for _, event := range newEvents {
		event.UID = EventID(event)
		merged = append(merged, event)
		seenUIDs[event.UID] = true
	}

	for uid, event := range existing {
		if seenUIDs[uid] {
			continue
		}
		if _, authoritative := authoritativeSources[event.Source]; authoritative && event.DTStart.After(now) {
			continue
		}
		merged = append(merged, event)
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
