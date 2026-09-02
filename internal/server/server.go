package server

import (
	"fmt"
	"net/http"

	"github.com/ciel-shieru/sit-ics-go/internal/ics"
)

type Server struct {
	port int
	cache *ics.ICSCache
}

func NewServer(port int, cache *ics.ICSCache) *Server {
	return &Server{
		port:  port,
		cache: cache,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/timetable.ics", func(w http.ResponseWriter, r *http.Request) {
		data := s.cache.Get()
		if len(data) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("Content-Disposition", `attachment; filename="timetable.ics"`)
		w.Write(data)
	})

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
