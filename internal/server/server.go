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
	mainAlerts      []calendar.Alert
	onlineAlerts    []calendar.Alert
	campusAlerts    []calendar.Alert
	bsEventsAlerts  []calendar.Alert
	bsDropboxAlerts []calendar.Alert
}

func NewServer(port int, cache *calendar.ICSCache, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts []calendar.Alert) *Server {
	return &Server{
		port:            port,
		cache:           cache,
		tz:              tz,
		refreshInterval: refreshInterval,
		mainAlerts:      mainAlerts,
		onlineAlerts:    onlineAlerts,
		campusAlerts:    campusAlerts,
		bsEventsAlerts:  bsEventsAlerts,
		bsDropboxAlerts: bsDropboxAlerts,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/timetable.ics", newTimetableHandler(s.cache, s.tz, s.refreshInterval, s.mainAlerts))
	http.HandleFunc("/timetable-online.ics", newOnlineHandler(s.cache, s.tz, s.refreshInterval, s.onlineAlerts))
	http.HandleFunc("/timetable-campus.ics", newCampusHandler(s.cache, s.tz, s.refreshInterval, s.campusAlerts))
	http.HandleFunc("/brightspace-events.ics", newBrightSpaceEventsHandler(s.cache, s.tz, s.refreshInterval, s.bsEventsAlerts))
	http.HandleFunc("/brightspace-dropbox.ics", newBrightSpaceDropboxHandler(s.cache, s.tz, s.refreshInterval, s.bsDropboxAlerts))

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
