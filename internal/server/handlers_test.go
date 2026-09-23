package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestNewXsiteQuizzesHandler(t *testing.T) {
	cache := calendar.NewICSCache()

	quizEvents := []calendar.Event{
		{
			Summary:     "[MOD1002] Quiz 1",
			Title:       "Quiz 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Sample Module Title Four",
			OrgUnitCode: "MOD1002",
			Source:      "brightspace-quizzes",
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	dropboxEvents := []calendar.Event{
		{
			Summary:     "[MOD1002] Assignment 1",
			Title:       "Assignment 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Sample Module Title Four",
			OrgUnitCode: "MOD1002",
			Source:      "brightspace-dropbox",
			DTStart:     time.Date(2026, 9, 16, 23, 59, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(quizEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := cache.Update(dropboxEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/xsite-quizzes.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !contains(body, "SUMMARY:[MOD1002] Quiz 1") {
		t.Error("Response should contain quiz event")
	}
	if contains(body, "SUMMARY:[MOD1002] Assignment 1") {
		t.Error("Response should not contain dropbox event")
	}
}

func TestNewXsiteQuizzesHandler_NoQuizzes(t *testing.T) {
	cache := calendar.NewICSCache()

	// Add only dropbox events (not quizzes)
	dropboxEvents := []calendar.Event{
		{
			Summary:     "[MOD1002] Assignment 1",
			Title:       "Assignment 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Sample Module Title Four",
			OrgUnitCode: "MOD1002",
			Source:      "brightspace-dropbox",
			DTStart:     time.Date(2026, 9, 16, 23, 59, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(dropboxEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/xsite-quizzes.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if contains(body, "BEGIN:VEVENT") {
		t.Error("Response should not contain any VEVENT (no quizzes in cache)")
	}
}

func TestNewXsiteQuizzesHandler_ContentHeaders(t *testing.T) {
	cache := calendar.NewICSCache()

	quizEvents := []calendar.Event{
		{
			Summary:     "[MOD1002] Quiz 1",
			Title:       "Quiz 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Sample Module Title Four",
			OrgUnitCode: "MOD1002",
			Source:      "brightspace-quizzes",
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(quizEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/xsite-quizzes.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/calendar" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/calendar")
	}

	contentDisposition := rr.Header().Get("Content-Disposition")
	if contentDisposition != `attachment; filename="xsite-quizzes.ics"` {
		t.Errorf("Content-Disposition = %q, want %q", contentDisposition, `attachment; filename="xsite-quizzes.ics"`)
	}
}

func TestNewXsiteHandler(t *testing.T) {
	cache := calendar.NewICSCache()

	events := []calendar.Event{
		{
			Summary:  "Campus Class",
			Location: "W1-05-07",
			DTStart:  time.Date(2026, 9, 7, 14, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
		{
			Summary:  "[COR2002] Lab 3 Due",
			Location: "",
			DTStart:  time.Date(2026, 9, 9, 23, 59, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 10, 23, 59, 0, 0, time.UTC),
			Source:   "brightspace-dropbox",
		},
		{
			Summary:     "[MOD1002] Quiz 1",
			Title:       "Quiz 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Sample Module Title Four",
			OrgUnitCode: "MOD1002",
			Source:      "brightspace-quizzes",
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !contains(body, "SUMMARY:[COR2001] Assignment 1") {
		t.Error("Response should contain brightspace-calendar event")
	}
	if !contains(body, "SUMMARY:[COR2002] Lab 3 Due") {
		t.Error("Response should contain brightspace-dropbox event")
	}
	if !contains(body, "SUMMARY:[MOD1002] Quiz 1") {
		t.Error("Response should contain brightspace-quizzes event")
	}
	if contains(body, "SUMMARY:Campus Class") {
		t.Error("Response should not contain non-brightspace event")
	}

	contentDisposition := rr.Header().Get("Content-Disposition")
	if contentDisposition != `attachment; filename="xsite.ics"` {
		t.Errorf("Content-Disposition = %q, want %q", contentDisposition, `attachment; filename="xsite.ics"`)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSub(s, substr)))
}

func findSub(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewTimetableHandler_ETagHeader(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header should be present on 200 response")
	}
	if !strings.HasPrefix(etag, `W/"`) {
		t.Errorf("ETag should be a weak validator, got: %q", etag)
	}
}

func TestNewTimetableHandler_IfNoneMatchMatch(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req1 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	etag := rr1.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d", rr2.Code, http.StatusNotModified)
	}
	if body := rr2.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
}

func TestNewTimetableHandler_IfNoneMatchNoMatch(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", `W/"non-matching-tag"`)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNewTimetableHandler_IfModifiedSinceCurrent(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	lastMod := cache.GetLastModified()
	if lastMod.IsZero() {
		t.Fatal("Expected non-zero lastModified")
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-Modified-Since", lastMod.UTC().Format(time.RFC1123))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusNotModified)
	}
}

func TestNewTimetableHandler_IfModifiedSincePast(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	pastTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-Modified-Since", pastTime.UTC().Format(time.RFC1123))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNewTimetableHandler_304IncludesHeaders(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	etag := rr.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Header().Get("ETag") == "" {
		t.Error("304 response should include ETag header")
	}
	if rr2.Header().Get("Cache-Control") == "" {
		t.Error("304 response should include Cache-Control header")
	}
}

func TestNewTimetableHandler_CacheControlHeader(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	cacheControl := rr.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Error("Cache-Control header should be present on 200 response")
	}
	if !strings.Contains(cacheControl, "max-age=") {
		t.Errorf("Cache-Control should contain max-age, got: %q", cacheControl)
	}
	if !strings.Contains(cacheControl, "immutable") {
		t.Errorf("Cache-Control should contain immutable, got: %q", cacheControl)
	}
}

func TestNewTimetableHandler_LastModifiedHeader(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	lastModified := rr.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Error("Last-Modified header should be present on 200 response")
	}
	_, err := time.Parse(time.RFC1123, lastModified)
	if err != nil {
		t.Errorf("Last-Modified should be in RFC 1123 format, got: %q, error: %v", lastModified, err)
	}
}

func TestNewTimetableHandler_MalformedIfNoneMatch(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", "not-a-valid-etag")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (malformed If-None-Match should be ignored)", rr.Code, http.StatusOK)
	}
}

func TestNewXsiteHandler_ConditionalRequests(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, false)

	req1 := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr1.Code, http.StatusOK)
	}
	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Error("ETag header should be present")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d", rr2.Code, http.StatusNotModified)
	}
	if body := rr2.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
}

func TestNewTimetableHandler_IfNoneMatchWildcard(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", "*")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (If-None-Match: * should match any existing resource)", rr.Code, http.StatusNotModified)
	}
	if body := rr.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
}

func TestNewTimetableHandler_IfNoneMatchWildcardWhitespace(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", " * ")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (whitespace around * should be tolerated)", rr.Code, http.StatusNotModified)
	}
}

func TestNewTimetableHandler_IfNoneMatchWeakComparison(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req1 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	etag := rr1.Header().Get("ETag")

	strongETag := strings.TrimPrefix(etag, `W/"`)
	strongETag = strings.TrimSuffix(strongETag, `"`)
	req2 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req2.Header.Set("If-None-Match", `"`+strongETag+`"`)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (weak comparison: W/\"...\" should match \"...\")", rr2.Code, http.StatusNotModified)
	}
}

func TestNewXsiteHandler_IfNoneMatchWildcard(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, false)

	req := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	req.Header.Set("If-None-Match", "*")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (xsite: If-None-Match: * should match)", rr.Code, http.StatusNotModified)
	}
	if body := rr.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
}

func TestNewTimetableHandler_IfNoneMatchMultipleETags(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	req1 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	etag := rr1.Header().Get("ETag")

	req2 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req2.Header.Set("If-None-Match", `W/"non-matching", `+etag)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (multiple ETags: second ETag matches → 304)", rr2.Code, http.StatusNotModified)
	}
}

func TestNewTimetableHandler_CombinedHeaders(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, false)

	// Request with both If-None-Match (non-matching) and If-Modified-Since (current)
	// Per RFC 7232 §6.8: If-None-Match doesn't match, but If-Modified-Since does → 304
	lastMod := cache.GetLastModified()
	req2 := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req2.Header.Set("If-None-Match", `W/"non-matching"`)
	req2.Header.Set("If-Modified-Since", lastMod.UTC().Format(time.RFC1123))
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotModified {
		t.Errorf("Status = %d, want %d (non-matching If-None-Match + current If-Modified-Since → 304)", rr2.Code, http.StatusNotModified)
	}
	if body := rr2.Body.String(); body != "" {
		t.Errorf("304 response body should be empty, got: %q", body)
	}
}

func TestNewTimetableHandler_DisableCaching_NoETag(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	etag := rr.Header().Get("ETag")
	if etag != "" {
		t.Errorf("ETag header should not be present when caching is disabled, got: %q", etag)
	}

	lastModified := rr.Header().Get("Last-Modified")
	if lastModified != "" {
		t.Errorf("Last-Modified header should not be present when caching is disabled, got: %q", lastModified)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("Cache-Control should contain no-cache when caching is disabled, got: %q", cacheControl)
	}
}

func TestNewTimetableHandler_DisableCaching_IgnoresIfNoneMatch(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-None-Match", `W/"any-etag"`)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (should return 200 even with If-None-Match when caching is disabled)", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Response body should not be empty when caching is disabled")
	}
}

func TestNewTimetableHandler_DisableCaching_IgnoresIfModifiedSince(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{Summary: "Test Event", DTStart: time.Date(2026, 9, 7, 9, 0, 0, 0, time.UTC), DTEnd: time.Date(2026, 9, 7, 11, 0, 0, 0, time.UTC)},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newTimetableHandler(cache, "Asia/Singapore", time.Hour, nil, true)

	lastMod := cache.GetLastModified()
	req := httptest.NewRequest(http.MethodGet, "/timetable.ics", nil)
	req.Header.Set("If-Modified-Since", lastMod.UTC().Format(time.RFC1123))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (should return 200 even with If-Modified-Since when caching is disabled)", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Response body should not be empty when caching is disabled")
	}
}

func TestNewXsiteHandler_DisableCaching_NoETag(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	etag := rr.Header().Get("ETag")
	if etag != "" {
		t.Errorf("ETag header should not be present when caching is disabled, got: %q", etag)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-cache") {
		t.Errorf("Cache-Control should contain no-cache when caching is disabled, got: %q", cacheControl)
	}
}

func TestNewXsiteHandler_DisableCaching_IgnoresIfNoneMatch(t *testing.T) {
	cache := calendar.NewICSCache()
	events := []calendar.Event{
		{
			Summary:  "[COR2001] Assignment 1",
			Location: "Online",
			DTStart:  time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC),
			Source:   "brightspace-calendar",
		},
	}
	if err := cache.Update(events, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newXsiteHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, true)

	req := httptest.NewRequest(http.MethodGet, "/xsite.ics", nil)
	req.Header.Set("If-None-Match", "*")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d (should return 200 even with If-None-Match: * when caching is disabled)", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if body == "" {
		t.Error("Response body should not be empty when caching is disabled")
	}
}
