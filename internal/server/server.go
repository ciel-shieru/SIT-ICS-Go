package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
)

type Server struct {
	port            int
	serverAddr      string
	cache           *calendar.ICSCache
	tz              string
	refreshInterval time.Duration
	mainAlerts      []calendar.Alert
	onlineAlerts    []calendar.Alert
	campusAlerts    []calendar.Alert
	bsEventsAlerts  []calendar.Alert
	bsDropboxAlerts []calendar.Alert
	bsQuizzesAlerts []calendar.Alert
}

func NewServer(port int, serverAddr string, cache *calendar.ICSCache, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts []calendar.Alert) *Server {
	return &Server{
		port:            port,
		serverAddr:      serverAddr,
		cache:           cache,
		tz:              tz,
		refreshInterval: refreshInterval,
		mainAlerts:      mainAlerts,
		onlineAlerts:    onlineAlerts,
		campusAlerts:    campusAlerts,
		bsEventsAlerts:  bsEventsAlerts,
		bsDropboxAlerts: bsDropboxAlerts,
		bsQuizzesAlerts: bsQuizzesAlerts,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/timetable.ics", newTimetableHandler(s.cache, s.tz, s.refreshInterval, s.mainAlerts))
	http.HandleFunc("/timetable-online.ics", newOnlineHandler(s.cache, s.tz, s.refreshInterval, s.onlineAlerts))
	http.HandleFunc("/timetable-campus.ics", newCampusHandler(s.cache, s.tz, s.refreshInterval, s.campusAlerts))
	http.HandleFunc("/xsite-events.ics", newXsiteEventsHandler(s.cache, s.tz, s.refreshInterval, s.bsEventsAlerts))
	http.HandleFunc("/xsite-dropbox.ics", newXsiteDropboxHandler(s.cache, s.tz, s.refreshInterval, s.bsDropboxAlerts))
	http.HandleFunc("/xsite-quizzes.ics", newXsiteQuizzesHandler(s.cache, s.tz, s.refreshInterval, s.bsQuizzesAlerts))
	http.HandleFunc("/xsite.ics", newXsiteHandler(s.cache, s.tz, s.refreshInterval, s.bsEventsAlerts, s.bsDropboxAlerts, s.bsQuizzesAlerts))

	addr := s.serverAddr
	if addr == "" {
		addr = config.DefaultServerAddr()
	}
	addr = fmt.Sprintf("%s:%d", addr, s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
