package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

type StatusResponse struct {
	LastFetchJobTime *time.Time        `json:"last_fetch_job_time"`
	JobStatus        string            `json:"job_status"`
	LastStatusIN4SIT string            `json:"last_status_in4sit"`
	LastStatusXsite  string            `json:"last_status_xsite"`
	ETags            map[string]string `json:"etags"`
	SHA256           map[string]string `json:"sha256"`
}

type statusOutput struct {
	path     string
	filename string
	data     []byte
}

// newStatusHandler reports the current in-memory representations. It never reads
// the persisted ICS outputs, so status checks do not cause output-file disk I/O.
func newStatusHandler(
	cache *calendar.ICSCache,
	tz string,
	refreshInterval time.Duration,
	mainAlerts, onlineAlerts, campusAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts []calendar.Alert,
	lastFetchJobTime func() time.Time,
	fetchStatuses func() (jobStatus, lastStatusIN4SIT, lastStatusXsite string),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		outputs, err := renderStatusOutputs(cache, tz, refreshInterval, mainAlerts, onlineAlerts, campusAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts)
		if err != nil {
			http.Error(w, "failed to render in-memory status", http.StatusInternalServerError)
			return
		}

		etags := make(map[string]string, len(outputs))
		hashes := make(map[string]string, len(outputs)-1)
		for _, output := range outputs {
			etags[output.path] = calendar.ComputeETag(output.data)
			if output.filename != "" {
				hash := sha256.Sum256(output.data)
				hashes[output.filename] = hex.EncodeToString(hash[:])
			}
		}

		var lastFetch *time.Time
		if lastFetchJobTime != nil {
			when := lastFetchJobTime()
			if !when.IsZero() {
				utc := when.UTC()
				lastFetch = &utc
			}
		}

		jobStatus, lastStatusIN4SIT, lastStatusXsite := "finished", "unknown", "unknown"
		if fetchStatuses != nil {
			jobStatus, lastStatusIN4SIT, lastStatusXsite = fetchStatuses()
		}

		payload := StatusResponse{
			LastFetchJobTime: lastFetch,
			JobStatus:        jobStatus,
			LastStatusIN4SIT: lastStatusIN4SIT,
			LastStatusXsite:  lastStatusXsite,
			ETags:            etags,
			SHA256:           hashes,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, "failed to encode status", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func renderStatusOutputs(
	cache *calendar.ICSCache,
	tz string,
	refreshInterval time.Duration,
	mainAlerts, onlineAlerts, campusAlerts, xsiteEventsAlerts, xsiteDropboxAlerts, xsiteQuizzesAlerts []calendar.Alert,
) ([]statusOutput, error) {
	mainData, err := cache.GetWithAlerts(tz, refreshInterval, mainAlerts)
	if err != nil {
		return nil, fmt.Errorf("render timetable.ics: %w", err)
	}
	onlineData, err := cache.GetFilteredWithAlerts(calendar.IsOnline, tz, refreshInterval, onlineAlerts)
	if err != nil {
		return nil, fmt.Errorf("render timetable-online.ics: %w", err)
	}
	campusData, err := cache.GetFilteredWithAlerts(calendar.IsCampus, tz, refreshInterval, campusAlerts)
	if err != nil {
		return nil, fmt.Errorf("render timetable-campus.ics: %w", err)
	}
	xsiteEventsData, err := cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, tz, refreshInterval, xsiteEventsAlerts)
	if err != nil {
		return nil, fmt.Errorf("render xsite-events.ics: %w", err)
	}
	xsiteDropboxData, err := cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, tz, refreshInterval, xsiteDropboxAlerts)
	if err != nil {
		return nil, fmt.Errorf("render xsite-dropbox.ics: %w", err)
	}
	xsiteQuizzesData, err := cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return e.Source == "brightspace-quizzes" }, tz, refreshInterval, xsiteQuizzesAlerts)
	if err != nil {
		return nil, fmt.Errorf("render xsite-quizzes.ics: %w", err)
	}
	xsiteData, err := cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return strings.HasPrefix(e.Source, "brightspace-") }, tz, refreshInterval, nil)
	if err != nil {
		return nil, fmt.Errorf("render xsite.ics: %w", err)
	}
	wakeyData, err := renderWakeyStatusJSON(cache, tz)
	if err != nil {
		return nil, fmt.Errorf("render Wakey JSON: %w", err)
	}

	return []statusOutput{
		{path: "/timetable.ics", filename: "timetable.ics", data: mainData},
		{path: "/timetable-online.ics", filename: "timetable-online.ics", data: onlineData},
		{path: "/timetable-campus.ics", filename: "timetable-campus.ics", data: campusData},
		{path: "/xsite-events.ics", filename: "xsite-events.ics", data: xsiteEventsData},
		{path: "/xsite-dropbox.ics", filename: "xsite-dropbox.ics", data: xsiteDropboxData},
		{path: "/xsite-quizzes.ics", filename: "xsite-quizzes.ics", data: xsiteQuizzesData},
		{path: "/xsite.ics", filename: "xsite.ics", data: xsiteData},
		{path: "/wakey-sitizen/timetable.json", data: wakeyData},
	}, nil
}

// renderWakeyStatusJSON renders the unfiltered JSON representation for the base
// Wakey timetable URL. It intentionally does not affect the handler's optional date filter.
func renderWakeyStatusJSON(cache *calendar.ICSCache, tz string) ([]byte, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}

	wakeyEvents := make([]WakeyEvent, 0)
	for _, e := range cache.GetEvents() {
		if strings.HasPrefix(e.Source, "brightspace-") {
			continue
		}
		eventType := "campus"
		if strings.EqualFold(e.Location, "Online") {
			eventType = "online"
		}
		wakeyEvents = append(wakeyEvents, WakeyEvent{
			CourseCode: e.CourseCode,
			Title:      e.Title,
			Summary:    e.Summary,
			Dtstart:    e.DTStart.In(loc).Format(time.RFC3339),
			Dtend:      e.DTEnd.In(loc).Format(time.RFC3339),
			Location:   e.Location,
			EventType:  eventType,
		})
	}

	return json.Marshal(WakeyResponse{
		SchemaVersion: wakeySchemaVersion,
		GeneratedAt:   cache.GetLastModified(),
		Timezone:      tz,
		Events:        wakeyEvents,
	})
}

type refreshResponse struct {
	Status string `json:"status"`
}

func newRefreshHandler(requestRefresh func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if requestRefresh == nil {
			http.Error(w, "refresh unavailable", http.StatusServiceUnavailable)
			return
		}

		requestRefresh()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(refreshResponse{Status: "scheduled"})
	}
}
