package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

func TestRunFetchReconcilesPastAndUpcomingEventsForSuccessfulSources(t *testing.T) {
	loc := testLoc(t)
	now := time.Now().In(loc)
	pastStart := now.Add(-48 * time.Hour)
	futureStart := now.Add(48 * time.Hour)

	seed := []calendar.Event{
		{Summary: "Past PeopleSoft", DTStart: pastStart, DTEnd: pastStart.Add(time.Hour)},
		{Summary: "Upcoming PeopleSoft", DTStart: futureStart, DTEnd: futureStart.Add(time.Hour)},
		{Summary: "Past BrightSpace Calendar", Source: "brightspace-calendar", DTStart: pastStart, DTEnd: pastStart.Add(time.Hour)},
		{Summary: "Upcoming BrightSpace Calendar", Source: "brightspace-calendar", DTStart: futureStart, DTEnd: futureStart.Add(time.Hour)},
		{Summary: "Past BrightSpace Dropbox", Source: "brightspace-dropbox", DTStart: pastStart, DTEnd: pastStart.Add(time.Hour)},
		{Summary: "Upcoming BrightSpace Dropbox", Source: "brightspace-dropbox", DTStart: futureStart, DTEnd: futureStart.Add(time.Hour)},
		{Summary: "Past BrightSpace Quiz", Source: "brightspace-quizzes", DTStart: pastStart, DTEnd: pastStart.Add(time.Hour)},
		{Summary: "Upcoming BrightSpace Quiz", Source: "brightspace-quizzes", DTStart: futureStart, DTEnd: futureStart.Add(time.Hour)},
	}

	cache := calendar.NewICSCache()
	cfg := newTestConfig(t, nil)
	if err := cache.Update(seed, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	currentPSStart := now.Add(72 * time.Hour)
	currentBSStart := now.Add(96 * time.Hour)
	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
			return []peoplesoft.Entry{{
				CourseCode: "ALT2501",
				ClassName:  "Current Fake Class",
				Section:    "A",
				Type:       "Tutorial",
				Day:        currentPSStart.Format("02/01/2006"),
				StartTime:  currentPSStart.Format("15:04"),
				EndTime:    currentPSStart.Add(time.Hour).Format("15:04"),
				Location:   "FAKE ROOM 1",
			}}, nil
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			entry := fakeBSEntry("Current Fake BrightSpace Event")
			entry.DTStart = currentBSStart.Format(time.RFC3339)
			entry.DTEnd = currentBSStart.Add(time.Hour).Format(time.RFC3339)
			return []brightspace.BrightSpaceStringEntry{entry}, nil
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			return nil, nil
		},
	}

	runFetch(context.Background(), cfg, auth.NewADFSProvider(mock), cache, loc)

	got := make(map[string]bool)
	for _, event := range cache.GetEvents() {
		got[event.Summary] = true
	}
	for _, summary := range []string{
		"Past PeopleSoft",
		"Past BrightSpace Calendar",
		"Past BrightSpace Dropbox",
		"Past BrightSpace Quiz",
		"Current Fake BrightSpace Event",
		"ALT2501 - A (Tutorial)",
	} {
		if !got[summary] && !(summary == "Current Fake BrightSpace Event" && hasSummaryContaining(cache.GetEvents(), summary)) {
			t.Errorf("expected event %q in cache", summary)
		}
	}
	for _, summary := range []string{
		"Upcoming PeopleSoft",
		"Upcoming BrightSpace Calendar",
		"Upcoming BrightSpace Dropbox",
		"Upcoming BrightSpace Quiz",
	} {
		if got[summary] {
			t.Errorf("expected upcoming deleted event %q to be removed", summary)
		}
	}
}

func TestRunFetchDoesNotReconcilePartiallyFetchedBrightSpaceSources(t *testing.T) {
	loc := testLoc(t)
	now := time.Now().In(loc)
	upcoming := now.Add(48 * time.Hour)
	seed := []calendar.Event{{
		Summary:  "Existing Upcoming BrightSpace Event",
		Source:   "brightspace-calendar",
		DTStart:  upcoming,
		DTEnd:    upcoming.Add(time.Hour),
	}}

	cache := calendar.NewICSCache()
	cfg := newTestConfig(t, nil)
	if err := cache.Update(seed, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
			return []peoplesoft.Entry{{
				CourseCode: "ALT2501",
				ClassName:  "Current Fake Class",
				Section:    "A",
				Type:       "Tutorial",
				Day:        upcoming.Add(24 * time.Hour).Format("02/01/2006"),
				StartTime:  "10:00",
				EndTime:    "11:00",
				Location:   "FAKE ROOM 1",
			}}, nil
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			partial := fakeBSEntry("Partial Course Event")
			return []brightspace.BrightSpaceStringEntry{partial}, errors.New("one course endpoint failed")
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			return nil, nil
		},
	}

	runFetch(context.Background(), cfg, auth.NewADFSProvider(mock), cache, loc)

	got := make(map[string]bool)
	for _, event := range cache.GetEvents() {
		got[event.Summary] = true
	}
	if !got["Existing Upcoming BrightSpace Event"] {
		t.Fatal("upcoming BrightSpace event was removed after a partial upstream fetch")
	}
	if !hasSummaryContaining(cache.GetEvents(), "Partial Course Event") {
		t.Error("partial BrightSpace result was not added to the cache")
	}
}

func hasSummaryContaining(events []calendar.Event, summary string) bool {
	for _, event := range events {
		if event.Summary == summary || containsText(event.Summary, summary) {
			return true
		}
	}
	return false
}

func containsText(value, fragment string) bool {
	return len(fragment) == 0 || (len(value) >= len(fragment) && strings.Contains(value, fragment))
}
