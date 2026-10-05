package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestNewWakeySitizenHandler_Basic(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			CourseCode: "ALT2501",
			Summary:    "[ALT2501] Lecture 1",
			Title:      "Lecture 1",
			Location:   "W1-05-07",
			DTStart:    time.Date(2026, 9, 7, 1, 0, 0, 0, time.UTC),
			DTEnd:      time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC),
		},
		{
			CourseCode: "COR2001",
			Summary:    "[COR2001] Tutorial 2",
			Title:      "Tutorial 2",
			Location:   "Online",
			DTStart:    time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC),
			DTEnd:      time.Date(2026, 9, 7, 10, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[MOD1002] Quiz 1",
			Title:    "Quiz 1",
			Location: "Online",
			Source:   "brightspace-quizzes",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[MOD1002] Assignment 1 Due",
			Location: "Online",
			Source:   "brightspace-dropbox",
			DTStart:  time.Date(2026, 9, 9, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[MOD1002] Lecture Notes",
			Location: "Zoom Online Meeting",
			Source:   "brightspace-calendar",
			DTStart:  time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC),
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, false)

	req := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	var resp WakeyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v; body: %s", err, rr.Body.String())
	}

	if resp.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", resp.SchemaVersion)
	}
	if resp.Timezone != "Asia/Singapore" {
		t.Errorf("timezone = %q, want %q", resp.Timezone, "Asia/Singapore")
	}
	if resp.GeneratedAt.IsZero() {
		t.Error("generated_at is zero, want last fetch time")
	}
	if want := cache.GetLastModified(); !resp.GeneratedAt.Equal(want) {
		t.Errorf("generated_at = %v, want %v (cache last modified)", resp.GeneratedAt, want)
	}

	if len(resp.Events) != 2 {
		t.Fatalf("len(events) = %d, want 2 (brightspace-* sources must be excluded); body: %s", len(resp.Events), rr.Body.String())
	}

	if resp.Events[0].CourseCode != "ALT2501" {
		t.Errorf("events[0].course_code = %q, want %q", resp.Events[0].CourseCode, "ALT2501")
	}
	if resp.Events[0].Title != "Lecture 1" {
		t.Errorf("events[0].title = %q, want %q", resp.Events[0].Title, "Lecture 1")
	}
	if resp.Events[0].Summary != "[ALT2501] Lecture 1" {
		t.Errorf("events[0].summary = %q, want %q", resp.Events[0].Summary, "[ALT2501] Lecture 1")
	}
	if resp.Events[0].Location != "W1-05-07" {
		t.Errorf("events[0].location = %q, want %q", resp.Events[0].Location, "W1-05-07")
	}
	if resp.Events[0].EventType != "campus" {
		t.Errorf("events[0].event_type = %q, want %q", resp.Events[0].EventType, "campus")
	}
	if resp.Events[0].Dtstart != "2026-09-07T09:00:00+08:00" {
		t.Errorf("events[0].dtstart = %q, want %q (RFC 3339, +08:00)", resp.Events[0].Dtstart, "2026-09-07T09:00:00+08:00")
	}
	if resp.Events[0].Dtend != "2026-09-07T11:00:00+08:00" {
		t.Errorf("events[0].dtend = %q, want %q (RFC 3339, +08:00)", resp.Events[0].Dtend, "2026-09-07T11:00:00+08:00")
	}

	if resp.Events[1].CourseCode != "COR2001" {
		t.Errorf("events[1].course_code = %q, want %q", resp.Events[1].CourseCode, "COR2001")
	}
	if resp.Events[1].EventType != "online" {
		t.Errorf("events[1].event_type = %q, want %q (Location=Online)", resp.Events[1].EventType, "online")
	}

	body := rr.Body.String()
	for _, blocked := range []string{"MOD1002", "brightspace"} {
		if strings.Contains(body, blocked) {
			t.Errorf("response body must not contain %q (brightspace events excluded)", blocked)
		}
	}
}

func TestNewWakeySitizenHandler_EmptyCache(t *testing.T) {
	cache := calendar.NewICSCache()

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, false)

	req := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d (empty result is 200, not 204)", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"events":[]`) {
		t.Errorf("body = %q, want it to contain %q (empty array, not null)", body, `"events":[]`)
	}
	if !strings.Contains(body, `"schema_version":1`) {
		t.Errorf("body = %q, want it to contain %q", body, `"schema_version":1`)
	}
	if !strings.Contains(body, `"timezone":"Asia/Singapore"`) {
		t.Errorf("body = %q, want it to contain %q", body, `"timezone":"Asia/Singapore"`)
	}

	var resp WakeyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if resp.Events == nil {
		t.Error("events is nil, want empty (non-nil) slice")
	}
}

func TestNewWakeySitizenHandler_SortedByDtstart(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:   "Later Event",
			Location:  "Room 102",
			DTStart:   time.Date(2026, 9, 8, 1, 0, 0, 0, time.UTC),
			DTEnd:     time.Date(2026, 9, 8, 3, 0, 0, 0, time.UTC),
		},
		{
			Summary:   "Earlier Event",
			Location:  "Room 101",
			DTStart:   time.Date(2026, 9, 7, 1, 0, 0, 0, time.UTC),
			DTEnd:     time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC),
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, false)

	req := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var resp WakeyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(resp.Events))
	}
	if resp.Events[0].Summary != "Earlier Event" {
		t.Errorf("events[0].summary = %q, want %q (deterministic DTStart ordering)", resp.Events[0].Summary, "Earlier Event")
	}
}

func TestNewWakeySitizenHandler_ETagAndConditional(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:   "Test Event",
			Location:  "Room 101",
			DTStart:   time.Date(2026, 9, 7, 1, 0, 0, 0, time.UTC),
			DTEnd:     time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC),
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, false)

	req1 := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", rr1.Code, http.StatusOK)
	}
	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag header should be present on 200 response")
	}
	if !strings.HasPrefix(etag, `W/"`) {
		t.Errorf("ETag should be a weak validator, got: %q", etag)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (If-None-Match with matching ETag)", rr2.Code, http.StatusNotModified)
	}
	if body := rr2.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
	if rr2.Header().Get("ETag") == "" {
		t.Error("304 response should include ETag header")
	}
	if rr2.Header().Get("Cache-Control") == "" {
		t.Error("304 response should include Cache-Control header")
	}
}

func TestNewWakeySitizenHandler_CacheControlHeader(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:   "Test Event",
			Location:  "Room 101",
			DTStart:   time.Date(2026, 9, 7, 1, 0, 0, 0, time.UTC),
			DTEnd:     time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC),
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, false)

	req := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "max-age=") {
		t.Errorf("Cache-Control should contain max-age, got: %q", cacheControl)
	}
	if !strings.Contains(cacheControl, "immutable") {
		t.Errorf("Cache-Control should contain immutable, got: %q", cacheControl)
	}
	if rr.Header().Get("Last-Modified") == "" {
		t.Error("Last-Modified header should be present on 200 response")
	}
}

func TestNewWakeySitizenHandler_DisableCaching(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:   "Test Event",
			Location:  "Room 101",
			DTStart:   time.Date(2026, 9, 7, 1, 0, 0, 0, time.UTC),
			DTEnd:     time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC),
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newWakeySitizenHandler(cache, "Asia/Singapore", time.Hour, true)

	req := httptest.NewRequest(http.MethodGet, "/wakey-sitizen/timetable.json", nil)
	req.Header.Set("If-None-Match", `W/"any-etag"`)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d (caching disabled: conditional headers ignored)", rr.Code, http.StatusOK)
	}
	if rr.Header().Get("ETag") != "" {
		t.Errorf("ETag header should not be present when caching is disabled, got: %q", rr.Header().Get("ETag"))
	}
	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("Cache-Control should contain no-cache when caching is disabled, got: %q", cacheControl)
	}
	if rr.Body.String() == "" {
		t.Error("Response body should not be empty when caching is disabled")
	}
}
