package calendar

import (
	"testing"
	"time"
)

func TestUpdateWithSourceReconciliationPreservesHistoryAndDeletesUpcoming(t *testing.T) {
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)
	event := func(summary, source string, start, end time.Time) Event {
		return Event{Summary: summary, Source: source, DTStart: start, DTEnd: end}
	}

	seed := []Event{
		event("Past PeopleSoft", "", now.Add(-2*time.Hour), now.Add(-time.Hour)),
		event("In-progress PeopleSoft", "", now.Add(-time.Minute), now.Add(time.Hour)),
		event("Upcoming PeopleSoft", "", now.Add(time.Hour), now.Add(2*time.Hour)),
		event("Past BrightSpace Calendar", "brightspace-calendar", now.Add(-3*time.Hour), now.Add(-2*time.Hour)),
		event("Upcoming BrightSpace Calendar", "brightspace-calendar", now.Add(2*time.Hour), now.Add(3*time.Hour)),
		event("Upcoming BrightSpace Dropbox", "brightspace-dropbox", now.Add(3*time.Hour), now.Add(4*time.Hour)),
		event("Upcoming BrightSpace Quiz", "brightspace-quizzes", now.Add(4*time.Hour), now.Add(5*time.Hour)),
	}

	cache := NewICSCache()
	if err := cache.Update(seed, "UTC"); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	current := event("Current Upstream Event", "", now.Add(5*time.Hour), now.Add(6*time.Hour))
	if err := cache.UpdateWithSourceReconciliation(
		[]Event{current},
		[]string{"", "brightspace-calendar", "brightspace-dropbox"},
		now,
	); err != nil {
		t.Fatalf("UpdateWithSourceReconciliation: %v", err)
	}

	got := make(map[string]bool)
	for _, cached := range cache.GetEvents() {
		got[cached.Summary] = true
	}

	for _, summary := range []string{
		"Past PeopleSoft",
		"In-progress PeopleSoft",
		"Past BrightSpace Calendar",
		"Upcoming BrightSpace Quiz", // Its source was not part of this successful fetch.
		"Current Upstream Event",
	} {
		if !got[summary] {
			t.Errorf("expected %q to remain in cache", summary)
		}
	}

	for _, summary := range []string{
		"Upcoming PeopleSoft",
		"Upcoming BrightSpace Calendar",
		"Upcoming BrightSpace Dropbox",
	} {
		if got[summary] {
			t.Errorf("expected upcoming deleted event %q to be removed", summary)
		}
	}
}

func TestUpdateWithSourceReconciliationRetainsEventsWhenNoSourceIsAuthoritative(t *testing.T) {
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)
	seed := []Event{{
		Summary:  "Unfetched Upcoming Event",
		Source:   "brightspace-calendar",
		DTStart:  now.Add(time.Hour),
		DTEnd:    now.Add(2 * time.Hour),
	}}

	cache := NewICSCache()
	if err := cache.Update(seed, "UTC"); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	if err := cache.UpdateWithSourceReconciliation(nil, nil, now); err != nil {
		t.Fatalf("UpdateWithSourceReconciliation: %v", err)
	}
	if got := cache.GetEvents(); len(got) != 1 {
		t.Fatalf("event count = %d, want 1", len(got))
	}
}
