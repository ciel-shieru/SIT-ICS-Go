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

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, "xsite.ics"))
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

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.Write(data)
	}
}
