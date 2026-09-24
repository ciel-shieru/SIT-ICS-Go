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
	trustedProxies  string
	disableCaching  bool
}

func NewServer(port int, serverAddr string, cache *calendar.ICSCache, tz string, refreshInterval time.Duration, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts []calendar.Alert, trustedProxies string, disableCaching bool) *Server {
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
		trustedProxies:  trustedProxies,
		disableCaching:  disableCaching,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/timetable.ics", newTimetableHandler(s.cache, s.tz, s.refreshInterval, s.mainAlerts, s.disableCaching))
	mux.HandleFunc("/timetable-online.ics", newOnlineHandler(s.cache, s.tz, s.refreshInterval, s.onlineAlerts, s.disableCaching))
	mux.HandleFunc("/timetable-campus.ics", newCampusHandler(s.cache, s.tz, s.refreshInterval, s.campusAlerts, s.disableCaching))
	mux.HandleFunc("/xsite-events.ics", newXsiteEventsHandler(s.cache, s.tz, s.refreshInterval, s.bsEventsAlerts, s.disableCaching))
	mux.HandleFunc("/xsite-dropbox.ics", newXsiteDropboxHandler(s.cache, s.tz, s.refreshInterval, s.bsDropboxAlerts, s.disableCaching))
	mux.HandleFunc("/xsite-quizzes.ics", newXsiteQuizzesHandler(s.cache, s.tz, s.refreshInterval, s.bsQuizzesAlerts, s.disableCaching))
	mux.HandleFunc("/xsite.ics", newXsiteHandler(s.cache, s.tz, s.refreshInterval, s.bsEventsAlerts, s.bsDropboxAlerts, s.bsQuizzesAlerts, s.disableCaching))

	handler, err := NewLoggingMiddleware(mux, s.trustedProxies)
	if err != nil {
		return fmt.Errorf("create logging middleware: %w", err)
	}

	addr := s.serverAddr
	if addr == "" {
		addr = config.DefaultServerAddr()
	}
	addr = fmt.Sprintf("%s:%d", addr, s.port)
	fmt.Printf("server: starting on %s\n", addr)
	return http.ListenAndServe(addr, handler)
}
