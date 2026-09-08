package server

import (
	"net/http"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func newTimetableHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := cache.Get(tz, refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable.ics"`)
		w.Write(data)
	}
}

func newOnlineHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := cache.GetFiltered(tz, calendar.IsOnline, refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-online.ics"`)
		w.Write(data)
	}
}

func newCampusHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := cache.GetFiltered(tz, calendar.IsCampus, refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-campus.ics"`)
		w.Write(data)
	}
}

func newXSiteEventsHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := cache.GetFiltered(tz, func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="xsite-events.ics"`)
		w.Write(data)
	}
}

func newXSiteDropboxHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := cache.GetFiltered(tz, func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="xsite-dropbox.ics"`)
		w.Write(data)
	}
}
