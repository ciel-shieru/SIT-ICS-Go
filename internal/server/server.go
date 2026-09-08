package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
)

type Server struct {
	port            int
	cache           *calendar.ICSCache
	tz              string
	refreshInterval time.Duration
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
	http.HandleFunc("/timetable.ics", newTimetableHandler(s.cache, s.tz, s.refreshInterval))
	http.HandleFunc("/timetable-online.ics", newOnlineHandler(s.cache, s.tz, s.refreshInterval))
	http.HandleFunc("/timetable-campus.ics", newCampusHandler(s.cache, s.tz, s.refreshInterval))
	http.HandleFunc("/xsite-events.ics", newXSiteEventsHandler(s.cache, s.tz, s.refreshInterval))
	http.HandleFunc("/xsite-dropbox.ics", newXSiteDropboxHandler(s.cache, s.tz, s.refreshInterval))

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
