package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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

	ps := peoplesoft.NewClient("https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/", cfg.ProxyURL, cfg.BrowserDebug, cfg.PeopleSoftLogResponse)

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
				AuthTimeout:       5 * time.Minute,
			})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
	case config.BrowserRemote:
		authBrowser, err = browser.NewRemoteBrowser(browser.BrowserConfig{
			ControlURL:        cfg.BrowserControlURL,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       5 * time.Minute,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
	default:
		log.Fatalf("unsupported browser mode: %s", cfg.BrowserMode)
	}

	provider := auth.NewADFSProvider(authBrowser)
	fetchAndUpdate := func() {
		log.Printf("scheduler: starting timetable fetch")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		authResult, err := provider.Authenticate(ctx, auth.AuthRequest{
			Username:        cfg.Username,
			Password:        cfg.Password,
			TOTPSecret:      cfg.TOTPSecret,
			PeopleSoftURL:   "https://in4sit.singaporetech.edu.sg/",
		})
		if err != nil {
			log.Printf("scheduler: auth failed: %v", err)
			return
		}

		ps.SetCookies(authResult.Cookies)
		log.Printf("scheduler: auth successful, %d cookies set, fetching timetable", len(authResult.Cookies))

		var allEntries []peoplesoft.Entry
		startDate := cfg.StartDate
		if startDate.IsZero() {
			startDate = time.Now().In(loc)
		}
		endDate := cfg.EndDate
		if endDate.IsZero() {
			endDate = startDate.AddDate(0, 0, 90)
		}

		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 7) {
			weekDate := d.Format("2006-01-02")
			entries, err := ps.FetchTimetable(ctx, weekDate)
			if err != nil {
				log.Printf("scheduler: fetch failed for %s: %v", weekDate, err)
				continue
			}
			allEntries = append(allEntries, entries...)
		}

		icsEvents := make([]ics.Event, 0, len(allEntries))
		for _, entry := range allEntries {
			icsEvents = append(icsEvents, ics.Event{
				CourseCode: entry.CourseCode,
				Summary:    fmt.Sprintf("%s - %s (%s)", entry.CourseCode, entry.Section, entry.Type),
				Location:   entry.Location,
				Description: fmt.Sprintf("Course: %s\nSection: %s\nType: %s", entry.CourseCode, entry.Section, entry.Type),
			})
		}

		if err := cache.Update(icsEvents, cfg.TZ); err != nil {
			log.Printf("scheduler: update failed: %v", err)
			return
		}

		if err := cache.SaveToFile(cfg.ICSStoragePath); err != nil {
			log.Printf("scheduler: save failed: %v", err)
			return
		}

		log.Printf("scheduler: fetch complete, %d events cached", len(icsEvents))
	}

	sched, err := scheduler.New(cfg.TZ)
	if err != nil {
		log.Fatalf("scheduler: %v", err)
	}
	sched.Start()

	if err := sched.AddJob(cfg.FetchCron, fetchAndUpdate); err != nil {
		log.Printf("scheduler: %v", err)
	}

	go func() {
		fetchAndUpdate()
	}()

	srv := server.NewServer(cfg.ServerPort, cache)

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

	if err := cache.SaveToFile(cfg.ICSStoragePath); err != nil {
		log.Printf("save on shutdown failed: %v", err)
	}
}
