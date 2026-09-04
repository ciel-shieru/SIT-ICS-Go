package server

import (
	"fmt"
	"net/http"

	"github.com/ciel-shieru/sit-ics-go/internal/ics"
)

type Server struct {
	port int
	cache *ics.ICSCache
	tz    string
}

func NewServer(port int, cache *ics.ICSCache, tz string) *Server {
	return &Server{
		port:  port,
		cache: cache,
		tz:    tz,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/timetable.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.Get(s.tz)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/timetable-online.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, ics.IsOnline)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-online.ics"`)
		w.Write(data)
	})

	http.HandleFunc("/timetable-campus.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.GetFiltered(s.tz, ics.IsNotOnline)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable-campus.ics"`)
		w.Write(data)
	})

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
