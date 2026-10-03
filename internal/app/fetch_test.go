package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
)

func testLoc(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return loc
}

func newTestConfig(t *testing.T, mutate func(*config.Config)) *config.Config {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{
		Username:           "test-user",
		Password:           "test-password",
		TOTPSecret:         "FAKETOTPSECRET",
		TZ:                 "Asia/Singapore",
		ICSRefreshInterval: time.Hour,
		XsiteEnabled:       true,
		ICSStoragePath:     filepath.Join(dir, "timetable.ics"),
		ICSOnlinePath:      filepath.Join(dir, "timetable-online.ics"),
		ICSCampusPath:      filepath.Join(dir, "timetable-campus.ics"),
		XsiteEventsPath:    filepath.Join(dir, "xsite-events.ics"),
		XsiteDropboxPath:   filepath.Join(dir, "xsite-dropbox.ics"),
		XsiteQuizzesPath:   filepath.Join(dir, "xsite-quizzes.ics"),
		XsitePath:          filepath.Join(dir, "xsite.ics"),
	}
	if mutate != nil {
		mutate(cfg)
	}
	return cfg
}

func seededPSEvent(loc *time.Location) calendar.Event {
	dtStart := time.Date(2026, 10, 5, 10, 0, 0, 0, loc)
	return calendar.Event{
		CourseCode:  "ALT2501",
		Summary:     "ALT2501 - A (Tutorial)",
		Location:    "FAKE HALL 1",
		Description: "Course: ALT2501\nClass: Fake Class\nSection: A\nType: Tutorial",
		DTStart:     dtStart,
		DTEnd:       dtStart.Add(time.Hour),
	}
}

func fakeBSEntry(title string) brightspace.BrightSpaceStringEntry {
	return brightspace.BrightSpaceStringEntry{
		Title:           title,
		OrgUnitId:       "ou-1",
		OrgUnitName:     "ALT2501 Fake Course",
		OrgUnitCode:     "ALT2501",
		Location:        "Online",
		Description:     "fake description",
		DTStart:         "2026-10-05T12:00:00+08:00",
		DTEnd:           "2026-10-05T13:00:00+08:00",
		Source:          "brightspace-calendar",
		CalendarEventID: 1,
	}
}

func TestPeopleSoftFetchFailureContinuesBrightSpace(t *testing.T) {
	loc := testLoc(t)

	var bsCalls, quizAPICalls int
	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) (string, error) {
			return "", fmt.Errorf("peoplesoft list page flake: %w", browser.ErrNavigation)
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			bsCalls++
			return []brightspace.BrightSpaceStringEntry{fakeBSEntry("Fake BS Event")}, nil
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			quizAPICalls++
			return nil, nil
		},
	}
	provider := auth.NewADFSProvider(mock)
	cfg := newTestConfig(t, nil)
	cache := calendar.NewICSCache()
	seed := seededPSEvent(loc)
	if err := cache.Update([]calendar.Event{seed}, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	runFetch(context.Background(), cfg, provider, cache, loc)

	if bsCalls == 0 {
		t.Error("brightspace fetch was not executed after peoplesoft failure")
	}
	if quizAPICalls == 0 {
		t.Error("quiz api fetch was not executed after peoplesoft failure")
	}

	data := string(cache.Get(cfg.TZ, cfg.ICSRefreshInterval))
	if !strings.Contains(data, "ALT2501 - A (Tutorial)") {
		t.Error("seeded peoplesoft event not retained in cache")
	}
	if !strings.Contains(data, "FAKE HALL 1") {
		t.Error("seeded peoplesoft event location not retained in cache")
	}
	if !strings.Contains(data, "Fake BS Event") {
		t.Error("brightspace event missing from cache")
	}

	// All 7 output files must be written on a degraded (peoplesoft-failed)
	// cycle, not just the main ICS.
	outputPaths := []string{
		cfg.ICSStoragePath,
		cfg.ICSOnlinePath,
		cfg.ICSCampusPath,
		cfg.XsiteEventsPath,
		cfg.XsiteDropboxPath,
		cfg.XsiteQuizzesPath,
		cfg.XsitePath,
	}
	for _, p := range outputPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("output file not saved: %v", err)
		}
	}

	mainData, err := os.ReadFile(cfg.ICSStoragePath)
	if err != nil {
		t.Fatalf("main ics output not saved: %v", err)
	}
	if !strings.Contains(string(mainData), "ALT2501 - A (Tutorial)") {
		t.Error("main ics output missing retained peoplesoft event")
	}
	if !strings.Contains(string(mainData), "Fake BS Event") {
		t.Error("main ics output missing brightspace event")
	}
}

func TestPeopleSoftFailureBlocklistDoesNotDeletePeopleSoft(t *testing.T) {
	loc := testLoc(t)

	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) (string, error) {
			return "", fmt.Errorf("peoplesoft list page flake: %w", browser.ErrNavigation)
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			return []brightspace.BrightSpaceStringEntry{fakeBSEntry("Fake BS Event")}, nil
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			return nil, nil
		},
	}
	provider := auth.NewADFSProvider(mock)
	cfg := newTestConfig(t, func(c *config.Config) {
		c.XsiteEventLocationBlocklist = "BLOCKED HALL"
	})
	cache := calendar.NewICSCache()

	ps := seededPSEvent(loc)
	ps.Location = "BLOCKED HALL"
	bsBlocked := calendar.Event{
		Summary:         "Blocked BS Event",
		Title:           "Blocked BS Event",
		OrgUnitID:       "ou-2",
		OrgUnitName:     "ALT2502 Fake Course",
		OrgUnitCode:     "ALT2502",
		Location:        "BLOCKED HALL",
		Description:     "fake description",
		DTStart:         time.Date(2026, 10, 5, 14, 0, 0, 0, loc),
		DTEnd:           time.Date(2026, 10, 5, 15, 0, 0, 0, loc),
		Source:          "brightspace-calendar",
		CalendarEventID: 2,
	}
	if err := cache.Update([]calendar.Event{ps, bsBlocked}, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	runFetch(context.Background(), cfg, provider, cache, loc)

	data := string(cache.Get(cfg.TZ, cfg.ICSRefreshInterval))
	if !strings.Contains(data, "ALT2501 - A (Tutorial)") {
		t.Error("peoplesoft event deleted by blocklist during degraded cycle")
	}
	if strings.Contains(data, "Blocked BS Event") {
		t.Error("blocked brightspace event was not removed during degraded cycle")
	}
}

func TestSmartMergeSkippedOnPeopleSoftFailure(t *testing.T) {
	loc := testLoc(t)

	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) (string, error) {
			return "", fmt.Errorf("peoplesoft list page flake: %w", browser.ErrNavigation)
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			// Control event: same module and overlapping time, but its location
			// lacks "Zoom Online Meeting", so smart merge must not consume it.
			online := fakeBSEntry("Fake BS Online Event")
			online.CalendarEventID = 2
			online.Location = "Online"
			online.Description = "no zoom link here"
			online.DTStart = "2026-10-05T10:30:00+08:00"
			online.DTEnd = "2026-10-05T11:30:00+08:00"

			// Matching event: same module, overlapping time, "Zoom Online
			// Meeting" location, and a real (fake) zoom URL in the description.
			// A non-skipped smart merge would consume this event and inject the
			// zoom details into the PeopleSoft event.
			zoom := fakeBSEntry("Fake BS Zoom Event")
			zoom.CalendarEventID = 3
			zoom.Location = "Zoom Online Meeting"
			zoom.Description = "join via https://fake-sg.zoom.us/j/12345678901"
			zoom.DTStart = "2026-10-05T10:30:00+08:00"
			zoom.DTEnd = "2026-10-05T11:30:00+08:00"

			return []brightspace.BrightSpaceStringEntry{online, zoom}, nil
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			return nil, nil
		},
	}
	provider := auth.NewADFSProvider(mock)
	cfg := newTestConfig(t, func(c *config.Config) {
		c.XsiteSmartMergeEnabled = true
	})
	cache := calendar.NewICSCache()
	seed := seededPSEvent(loc)
	seed.Location = "Online"
	if err := cache.Update([]calendar.Event{seed}, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	runFetch(context.Background(), cfg, provider, cache, loc)

	data := string(cache.Get(cfg.TZ, cfg.ICSRefreshInterval))
	if !strings.Contains(data, "Fake BS Zoom Event") {
		t.Error("brightspace zoom event not kept standalone")
	}

	psUID := calendar.EventID(seed)
	var psBlock string
	for _, b := range strings.Split(data, "BEGIN:VEVENT")[1:] {
		if strings.Contains(b, "UID:"+psUID) {
			psBlock = b
			break
		}
	}
	if psBlock == "" {
		t.Fatal("seeded peoplesoft event removed from cache")
	}
	if strings.Contains(psBlock, "fake-sg.zoom.us") || strings.Contains(psBlock, "Zoom Meeting Details") {
		t.Error("zoom details injected into peoplesoft event during degraded cycle")
	}
	if !strings.Contains(psBlock, "Course: ALT2501") {
		t.Error("peoplesoft event description modified during degraded cycle")
	}
}

func TestBrowserFatalOnPeopleSoftFailure(t *testing.T) {
	loc := testLoc(t)

	var bsCalls, quizAPICalls int
	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, nil
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) (string, error) {
			return "", fmt.Errorf("browser gone: %w", browser.ErrBrowserUnavailable)
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			bsCalls++
			return nil, nil
		},
		FetchBrightSpaceQuizzesAPIFunc: func(ctx context.Context, baseURL string) ([]brightspace.QuizAPI, error) {
			quizAPICalls++
			return nil, nil
		},
	}
	provider := auth.NewADFSProvider(mock)
	cfg := newTestConfig(t, nil)
	cache := calendar.NewICSCache()
	if err := cache.Update([]calendar.Event{seededPSEvent(loc)}, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	eventsBefore := cache.EventCount()

	runFetch(context.Background(), cfg, provider, cache, loc)

	if bsCalls != 0 {
		t.Error("brightspace fetch invoked on fatal browser failure")
	}
	if quizAPICalls != 0 {
		t.Error("quiz api fetch invoked on fatal browser failure")
	}
	if cache.EventCount() != eventsBefore {
		t.Error("cache modified on fatal browser failure")
	}
	if _, err := os.Stat(cfg.ICSStoragePath); !os.IsNotExist(err) {
		t.Error("outputs saved on fatal browser failure")
	}
}

func TestIsFatalPostAuth(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrBrowserUnavailable", browser.ErrBrowserUnavailable},
		{"ErrBrowserLaunch", browser.ErrBrowserLaunch},
		{"ErrBrowserConnect", browser.ErrBrowserConnect},
		{"ErrAuthentication", browser.ErrAuthentication},
		{"ErrAuthenticationTimeout", browser.ErrAuthenticationTimeout},
		{"ErrCredentialExtraction", browser.ErrCredentialExtraction},
	}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("some random failure"), false},
		{"wrapped plain error", fmt.Errorf("outer: %w", errors.New("inner failure")), false},
	}
	for _, s := range sentinels {
		tests = append(tests,
			struct {
				name string
				err  error
				want bool
			}{s.name + " bare", s.err, true},
			struct {
				name string
				err  error
				want bool
			}{s.name + " wrapped", fmt.Errorf("context: %w", s.err), true},
		)
	}
	tests = append(tests, struct {
		name string
		err  error
		want bool
	}{"ErrAuthentication double-wrapped", fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", browser.ErrAuthentication)), true})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFatalPostAuth(tt.err); got != tt.want {
				t.Errorf("isFatalPostAuth(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAuthFailureAbortsFetch(t *testing.T) {
	loc := testLoc(t)

	var ttCalls, bsCalls int
	mock := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, fmt.Errorf("launch: %w", browser.ErrBrowserLaunch)
		},
		FetchTimetableFunc: func(ctx context.Context, weekDate string) (string, error) {
			ttCalls++
			return "", nil
		},
		FetchBrightSpaceFunc: func(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
			bsCalls++
			return nil, nil
		},
	}
	provider := auth.NewADFSProvider(mock)
	cfg := newTestConfig(t, nil)
	cache := calendar.NewICSCache()
	if err := cache.Update([]calendar.Event{seededPSEvent(loc)}, cfg.TZ); err != nil {
		t.Fatalf("seed cache: %v", err)
	}

	runFetch(context.Background(), cfg, provider, cache, loc)

	if ttCalls != 0 {
		t.Error("timetable fetch invoked after auth failure")
	}
	if bsCalls != 0 {
		t.Error("brightspace fetch invoked after auth failure")
	}
}
