package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/ics"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
	"github.com/ciel-shieru/sit-ics-go/internal/scheduler"
	"github.com/ciel-shieru/sit-ics-go/internal/server"
)

const fetchTimeout = 5 * time.Minute

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("invalid timezone %q: %v", cfg.TZ, err)
	}
	time.Local = loc

	cache := ics.NewICSCache()
	if err := cache.LoadFromFile(cfg.ICSStoragePath); err != nil {
		log.Printf("ics: failed to load from disk: %v", err)
	}

	var authBrowser browser.AuthBrowser
	switch cfg.BrowserMode {
	case config.BrowserSystem, config.BrowserAuto, config.BrowserRod:
		authBrowser, err = browser.NewLocalBrowser(browser.BrowserConfig{
			Executable:        cfg.BrowserExecutable,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       fetchTimeout,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
	case config.BrowserRemote:
		authBrowser, err = browser.NewRemoteBrowser(browser.BrowserConfig{
			RemoteHost:        cfg.BrowserRemoteHost,
			RemotePort:        cfg.BrowserRemotePort,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       fetchTimeout,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
	default:
		log.Fatalf("unsupported browser mode: %s", cfg.BrowserMode)
	}

	provider := auth.NewADFSProvider(authBrowser)

	sched, err := scheduler.New(cfg.TZ)
	if err != nil {
		log.Fatalf("scheduler: %v", err)
	}
	sched.Start()

	go func() {
		runFetch(cfg, provider, cache, loc)
	}()

	if err := sched.AddJob(cfg.FetchCron, func() { runFetch(cfg, provider, cache, loc) }); err != nil {
		log.Printf("scheduler: %v", err)
	}

	srv := server.NewServer(cfg.ServerPort, cache, cfg.TZ)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	<-sigChan
	log.Println("shutting down...")

	sched.Stop()

	if authBrowser != nil {
		authBrowser.Close()
	}

	if err := cache.SaveAllToFiles(cfg.ICSStoragePath, cfg.ICSOnlinePath, cfg.ICSCampusPath, cfg.TZ); err != nil {
		log.Printf("save on shutdown failed: %v", err)
	}
}

func runFetch(cfg *config.Config, provider *auth.ADFSProvider, cache *ics.ICSCache, loc *time.Location) {
	log.Printf("scheduler: starting timetable fetch")
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	authResult, err := provider.Authenticate(ctx, auth.AuthRequest{
		Username:      cfg.Username,
		Password:      cfg.Password,
		TOTPSecret:    cfg.TOTPSecret,
		PeopleSoftURL: "https://in4sit.singaporetech.edu.sg/",
	})
	if err != nil {
		log.Printf("scheduler: auth failed: %v", err)
		return
	}

	log.Printf("scheduler: auth successful, %d cookies set, fetching timetable", len(authResult.Cookies))

	entries, err := provider.FetchTimetable(ctx, "")
	if err != nil {
		log.Printf("scheduler: fetch failed: %v", err)
		return
	}
	allEntries := entries

	icsEvents := make([]ics.Event, 0, len(allEntries))
	for _, entry := range allEntries {
		dtStart, dtEnd, err := parseEntryDateTime(entry, loc)
		if err != nil {
			log.Printf("scheduler: skipping entry %s: %v", entry.CourseCode, err)
			continue
		}
		icsEvents = append(icsEvents, ics.Event{
			CourseCode:  entry.CourseCode,
			Summary:     fmt.Sprintf("%s - %s (%s)", entry.CourseCode, entry.Section, entry.Type),
			Location:    entry.Location,
			Description: fmt.Sprintf("Course: %s\nClass: %s\nSection: %s\nType: %s", entry.CourseCode, entry.ClassName, entry.Section, entry.Type),
			DTStart:     dtStart,
			DTEnd:       dtEnd,
		})
	}

	if err := cache.Update(icsEvents, cfg.TZ); err != nil {
		log.Printf("scheduler: update failed: %v", err)
		return
	}

	if err := cache.SaveAllToFiles(cfg.ICSStoragePath, cfg.ICSOnlinePath, cfg.ICSCampusPath, cfg.TZ); err != nil {
		log.Printf("scheduler: save failed: %v", err)
		return
	}

	log.Printf("scheduler: fetch complete, %d events cached", len(icsEvents))
}

func parseEntryDateTime(entry peoplesoft.Entry, loc *time.Location) (time.Time, time.Time, error) {
	parts := strings.Split(entry.Day, "/")
	if len(parts) != 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid day format: %s", entry.Day)
	}
	var day, month, year int
	fmt.Sscanf(parts[0], "%d", &day)
	fmt.Sscanf(parts[1], "%d", &month)
	fmt.Sscanf(parts[2], "%d", &year)

	startHour, startMin := parseHoursMinutes(entry.StartTime)
	endHour, endMin := parseHoursMinutes(entry.EndTime)

	dtStart := time.Date(year, time.Month(month), day, startHour, startMin, 0, 0, loc)
	dtEnd := time.Date(year, time.Month(month), day, endHour, endMin, 0, 0, loc)
	return dtStart, dtEnd, nil
}

func parseHoursMinutes(s string) (int, int) {
	var h, m int
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h, m
}
