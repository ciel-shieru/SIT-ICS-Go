package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

func newTimetableHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, nil, "timetable.ics")
}

func newOnlineHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsOnline, "timetable-online.ics")
}

func newCampusHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, calendar.IsCampus, "timetable-campus.ics")
}

func newBrightSpaceEventsHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, "brightspace-events.ics")
}

func newBrightSpaceDropboxHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration) http.HandlerFunc {
	return newFilteredHandler(cache, tz, refreshInterval, func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, "brightspace-dropbox.ics")
}

func newFilteredHandler(cache *calendar.ICSCache, tz string, refreshInterval time.Duration, filterFn func(calendar.Event) bool, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var data []byte
		if filterFn == nil {
			data = cache.Get(tz, refreshInterval)
		} else {
			data = cache.GetFiltered(tz, filterFn, refreshInterval)
		}
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Write(data)
	}
}
