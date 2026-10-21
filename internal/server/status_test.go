package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func TestStatusHandlerReportsInMemoryHashesETagsAndLastFetchTime(t *testing.T) {
	cache := calendar.NewICSCache()
	if err := cache.Update([]calendar.Event{
		{
			CourseCode: "ALT2501",
			Summary:    "[ALT2501] Sample lecture",
			Title:      "Sample lecture",
			Location:   "Online",
			DTStart:    time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC),
			DTEnd:      time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC),
		},
		{
			Summary:  "Sample Xsite event",
			Source:   "brightspace-calendar",
			Location: "Online",
			DTStart:  time.Date(2026, 10, 22, 1, 0, 0, 0, time.UTC),
			DTEnd:    time.Date(2026, 10, 22, 2, 0, 0, 0, time.UTC),
		},
	}, "Asia/Singapore"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	lastFetch := time.Date(2026, 10, 21, 5, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	handler := newStatusHandler(cache, "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil, func() time.Time {
		return lastFetch
	}, func() (string, string, string) {
		return "running", "successful", "failed"
	})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	var response StatusResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode /status: %v", err)
	}
	if response.LastFetchJobTime == nil || !response.LastFetchJobTime.Equal(lastFetch) {
		t.Errorf("last_fetch_job_time = %v, want %v", response.LastFetchJobTime, lastFetch)
	}
	if response.JobStatus != "running" {
		t.Errorf("job_status = %q, want running", response.JobStatus)
	}
	if response.LastStatusIN4SIT != "successful" {
		t.Errorf("last_status_in4sit = %q, want successful", response.LastStatusIN4SIT)
	}
	if response.LastStatusXsite != "failed" {
		t.Errorf("last_status_xsite = %q, want failed", response.LastStatusXsite)
	}
	if len(response.SHA256) != 7 {
		t.Errorf("SHA256 output count = %d, want 7", len(response.SHA256))
	}
	if len(response.ETags) != 8 {
		t.Errorf("ETag count = %d, want 8", len(response.ETags))
	}

	mainData, err := cache.GetWithAlerts("Asia/Singapore", time.Hour, nil)
	if err != nil {
		t.Fatalf("render main ICS: %v", err)
	}
	if got, want := response.ETags["/timetable.ics"], calendar.ComputeETag(mainData); got != want {
		t.Errorf("timetable ETag = %q, want %q", got, want)
	}
	sum := sha256.Sum256(mainData)
	if got, want := response.SHA256["timetable.ics"], hex.EncodeToString(sum[:]); got != want {
		t.Errorf("timetable SHA256 = %q, want %q", got, want)
	}
	for name, hash := range response.SHA256 {
		if len(hash) != 64 {
			t.Errorf("SHA256[%q] length = %d, want 64", name, len(hash))
		}
		if _, err := hex.DecodeString(hash); err != nil {
			t.Errorf("SHA256[%q] is not hexadecimal: %v", name, err)
		}
	}
}

func TestStatusHandlerAllowsOnlyGET(t *testing.T) {
	handler := newStatusHandler(calendar.NewICSCache(), "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil, nil, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/status", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want GET", got)
	}
}

func TestStatusHandlerHasNullLastFetchBeforeFirstJob(t *testing.T) {
	handler := newStatusHandler(calendar.NewICSCache(), "Asia/Singapore", time.Hour, nil, nil, nil, nil, nil, nil, nil, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode /status: %v", err)
	}
	if value, ok := response["last_fetch_job_time"]; !ok || value != nil {
		t.Errorf("last_fetch_job_time = %#v, want null", value)
	}
}

func TestRefreshHandlerSchedulesPOSTOnly(t *testing.T) {
	called := 0
	handler := newRefreshHandler(func() { called++ })
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/refresh", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if called != 1 {
		t.Errorf("refresh callback calls = %d, want 1", called)
	}
	var response refreshResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode /refresh: %v", err)
	}
	if response.Status != "scheduled" {
		t.Errorf("status = %q, want scheduled", response.Status)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/refresh", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /refresh code = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodPost {
		t.Errorf("GET /refresh Allow = %q, want POST", got)
	}
	if called != 1 {
		t.Errorf("refresh callback calls after rejected GET = %d, want 1", called)
	}
}

func TestRefreshHandlerUnavailableWithoutCallback(t *testing.T) {
	handler := newRefreshHandler(nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/refresh", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
