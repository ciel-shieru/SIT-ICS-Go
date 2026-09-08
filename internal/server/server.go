package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

type Server struct {
	port              int
	cache             *calendar.ICSCache
	tz                string
	refreshInterval   time.Duration
}

func NewServer(port int, cache *calendar.ICSCache, tz string, refreshInterval time.Duration) *Server {
	return &Server{
		port:            port,
		cache:           cache,
		tz:              tz,
		refreshInterval: refreshInterval,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/timetable.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.Get(s.tz, s.refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/timetable-online.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, calendar.IsOnline, s.refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-online.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/timetable-campus.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, calendar.IsNotOnlineAndNotBrightSpace, s.refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-campus.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/xsite-events.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, func(e calendar.Event) bool { return e.Source == "brightspace-calendar" }, s.refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="xsite-events.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/xsite-dropbox.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, func(e calendar.Event) bool { return e.Source == "brightspace-dropbox" }, s.refreshInterval)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="xsite-dropbox.ics"`)
		w.Write(data)
	})

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
