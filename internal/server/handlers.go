package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func newTimetableHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, nil, "timetable.ics", alerts, disableCaching)
}

func newOnlineHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsOnline, "timetable-online.ics", alerts, disableCaching)
}

func newCampusHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsCampus, "timetable-campus.ics", alerts, disableCaching)
}

func newXsiteEventsHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, "xsite-events.ics", alerts, disableCaching)
}

func newXsiteDropboxHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, "xsite-dropbox.ics", alerts, disableCaching)
}

func newXsiteQuizzesHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-quizzes" }, "xsite-quizzes.ics", alerts, disableCaching)
}

func newXsiteHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, eventsAlerts, dropboxAlerts, quizzesAlerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		var err error
		data, err = cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return strings.HasPrefix(e.Source, "brightspace-") }, tz, refreshInterval, nil)
		if err != nil || len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, "xsite.ics"))
		if disableCaching {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			etag := calendar.ComputeETag(data)
			lastMod := cache.GetLastModified()

			if inm := r.Header.Get("If-None-Match"); inm != "" {
				if strings.TrimSpace(inm) == "*" {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
				serverETagHash := strings.TrimPrefix(etag, `W/"`)
				serverETagHash = strings.TrimSuffix(serverETagHash, `"`)
				if etagMatches(inm, serverETagHash) {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
			}

			if ims := r.Header.Get("If-Modified-Since"); ims != "" {
				if modTime, parseErr := time.Parse(time.RFC1123, ims); parseErr == nil {
					if !lastMod.IsZero() && !modTime.Add(time.Second).Before(lastMod) {
						writeNotModified(w, etag, lastMod, refreshInterval)
						return
					}
				}
			}

			w.Header().Set("ETag", etag)
			if !lastMod.IsZero() {
				w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
			}
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
		}
		w.Write(data)
	}
}

// wakeySchemaVersion is the JSON schema version served by the
// /wakey-sitizen/timetable.json endpoint. Bump when the schema changes;
// the client app validates it before deserializing.
const wakeySchemaVersion = 1

// WakeyResponse is the JSON payload served to the Wakey-SITizen companion app.
type WakeyResponse struct {
	SchemaVersion int          `json:"schema_version"`
	GeneratedAt   time.Time    `json:"generated_at"`
	Timezone      string       `json:"timezone"`
	Events        []WakeyEvent `json:"events"`
}

// WakeyEvent is a single timetable entry in the Wakey-SITizen JSON payload.
type WakeyEvent struct {
	CourseCode string `json:"course_code"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Dtstart    string `json:"dtstart"`
	Dtend      string `json:"dtend"`
	Location   string `json:"location"`
	EventType  string `json:"event_type"`
}

// newWakeySitizenHandler serves the timetable as JSON for the Wakey-SITizen
// companion app. Events sourced from BrightSpace (Source prefix "brightspace-")
// are excluded. Always returns 200 OK, including for an empty result set.
func newWakeySitizenHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, disableCaching bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		resp := WakeyResponse{
			SchemaVersion: wakeySchemaVersion,
			GeneratedAt:   cache.GetLastModified(),
			Timezone:      tz,
			Events:        wakeyEvents,
		}

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if disableCaching {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			etag := calendar.ComputeETag(data)
			lastMod := cache.GetLastModified()

			if inm := r.Header.Get("If-None-Match"); inm != "" {
				if strings.TrimSpace(inm) == "*" {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
				serverETagHash := strings.TrimPrefix(etag, `W/"`)
				serverETagHash = strings.TrimSuffix(serverETagHash, `"`)
				if etagMatches(inm, serverETagHash) {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
			}

			if ims := r.Header.Get("If-Modified-Since"); ims != "" {
				if modTime, parseErr := time.Parse(time.RFC1123, ims); parseErr == nil {
					if !lastMod.IsZero() && !modTime.Add(time.Second).Before(lastMod) {
						writeNotModified(w, etag, lastMod, refreshInterval)
						return
					}
				}
			}

			w.Header().Set("ETag", etag)
			if !lastMod.IsZero() {
				w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
			}
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}

func newFilteredHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, filterFn func(calendar.Event) bool, filename string, alerts []calendar.Alert, disableCaching bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		var err error
		if filterFn == nil {
			data, err = cache.GetWithAlerts(tz, refreshInterval, alerts)
		} else {
			data, err = cache.GetFilteredWithAlerts(filterFn, tz, refreshInterval, alerts)
		}
		if err != nil || len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		if !disableCaching {
			etag := calendar.ComputeETag(data)
			lastMod := cache.GetLastModified()

			if inm := r.Header.Get("If-None-Match"); inm != "" {
				if strings.TrimSpace(inm) == "*" {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
				serverETagHash := strings.TrimPrefix(etag, `W/"`)
				serverETagHash = strings.TrimSuffix(serverETagHash, `"`)
				if etagMatches(inm, serverETagHash) {
					writeNotModified(w, etag, lastMod, refreshInterval)
					return
				}
			}

			if ims := r.Header.Get("If-Modified-Since"); ims != "" {
				if modTime, parseErr := time.Parse(time.RFC1123, ims); parseErr == nil {
					if !lastMod.IsZero() && !modTime.Add(time.Second).Before(lastMod) {
						writeNotModified(w, etag, lastMod, refreshInterval)
						return
					}
				}
			}

			w.Header().Set("ETag", etag)
			if !lastMod.IsZero() {
				w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
			}
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
		} else {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		w.Write(data)
	}
}

// etagMatches checks if any ETag in the If-None-Match header value matches the server's ETag hash.
// Per RFC 7232 §2.3, If-None-Match can be a comma-separated list of entity-tags.
func etagMatches(headerValue, serverETagHash string) bool {
	if strings.TrimSpace(headerValue) == "*" {
		return true
	}

	for _, raw := range strings.Split(headerValue, ",") {
		raw = strings.TrimSpace(raw)
		if strings.HasPrefix(raw, "W/") {
			raw = strings.TrimSpace(raw[2:])
		}
		if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
			raw = raw[1 : len(raw)-1]
		}
		if raw == serverETagHash {
			return true
		}
	}
	return false
}

func writeNotModified(w http.ResponseWriter, etag string, lastMod time.Time, refreshInterval time.Duration) {
	w.Header().Set("ETag", etag)
	if !lastMod.IsZero() {
		w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
	}
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
	w.WriteHeader(http.StatusNotModified)
}
