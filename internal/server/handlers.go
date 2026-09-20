package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func newTimetableHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, nil, "timetable.ics", alerts)
}

func newOnlineHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsOnline, "timetable-online.ics", alerts)
}

func newCampusHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsCampus, "timetable-campus.ics", alerts)
}

func newXsiteEventsHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, "xsite-events.ics", alerts)
}

func newXsiteDropboxHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, "xsite-dropbox.ics", alerts)
}

func newXsiteQuizzesHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, alerts []calendar.Alert) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-quizzes" }, "xsite-quizzes.ics", alerts)
}

func newXsiteHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, eventsAlerts, dropboxAlerts, quizzesAlerts []calendar.Alert) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		var err error
		data, err = cache.GetFilteredWithAlerts(func(e calendar.Event) bool { return strings.HasPrefix(e.Source, "brightspace-") }, tz, refreshInterval, nil)
		if err != nil || len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, "xsite.ics"))
		w.Header().Set("ETag", etag)
		if !lastMod.IsZero() {
			w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
		w.Write(data)
	}
}

func newFilteredHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, filterFn func(calendar.Event) bool, filename string, alerts []calendar.Alert) http.HandlerFunc {
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

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Header().Set("ETag", etag)
		if !lastMod.IsZero() {
			w.Header().Set("Last-Modified", lastMod.UTC().Format(time.RFC1123))
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", int(refreshInterval.Seconds())))
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
