package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestNewBrightSpaceQuizzesHandler(t *testing.T) {
	cache := calendar.NewICSCache()

	quizEvents := []calendar.Event{
		{
			Summary:     "[SIT1001] Quiz 1",
			Title:       "Quiz 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Introduction to Computer Science",
			OrgUnitCode: "SIT1001",
			Source:      "brightspace-quizzes",
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	dropboxEvents := []calendar.Event{
		{
			Summary:     "[SIT1001] Assignment 1",
			Title:       "Assignment 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Introduction to Computer Science",
			OrgUnitCode: "SIT1001",
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

	handler := newBrightSpaceQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil)

	req := httptest.NewRequest(http.MethodGet, "/quizzes.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !contains(body, "SUMMARY:[SIT1001] Quiz 1") {
		t.Error("Response should contain quiz event")
	}
	if contains(body, "SUMMARY:[SIT1001] Assignment 1") {
		t.Error("Response should not contain dropbox event")
	}
}

func TestNewBrightSpaceQuizzesHandler_NoQuizzes(t *testing.T) {
	cache := calendar.NewICSCache()

	// Add only dropbox events (not quizzes)
	dropboxEvents := []calendar.Event{
		{
			Summary:     "[SIT1001] Assignment 1",
			Title:       "Assignment 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Introduction to Computer Science",
			OrgUnitCode: "SIT1001",
			Source:      "brightspace-dropbox",
			DTStart:     time.Date(2026, 9, 16, 23, 59, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 17, 23, 59, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(dropboxEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newBrightSpaceQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil)

	req := httptest.NewRequest(http.MethodGet, "/quizzes.ics", nil)
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

func TestNewBrightSpaceQuizzesHandler_ContentHeaders(t *testing.T) {
	cache := calendar.NewICSCache()

	quizEvents := []calendar.Event{
		{
			Summary:     "[SIT1001] Quiz 1",
			Title:       "Quiz 1",
			OrgUnitID:   "12345",
			OrgUnitName: "Introduction to Computer Science",
			OrgUnitCode: "SIT1001",
			Source:      "brightspace-quizzes",
			DTStart:     time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
			DTEnd:       time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC),
		},
	}

	if err := cache.Update(quizEvents, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	handler := newBrightSpaceQuizzesHandler(cache, "Asia/Singapore", time.Hour, nil)

	req := httptest.NewRequest(http.MethodGet, "/quizzes.ics", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/calendar" {
		t.Errorf("Content-Type = %q, want %q", contentType, "text/calendar")
	}

	contentDisposition := rr.Header().Get("Content-Disposition")
	if contentDisposition != `attachment; filename="brightspace-quizzes.ics"` {
		t.Errorf("Content-Disposition = %q, want %q", contentDisposition, `attachment; filename="brightspace-quizzes.ics"`)
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
