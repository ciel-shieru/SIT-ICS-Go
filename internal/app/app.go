package app

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/ics"
	"github.com/ciel-shieru/sit-ics-go/internal/scheduler"
	"github.com/ciel-shieru/sit-ics-go/internal/server"
)

const FetchTimeout = 5 * time.Minute

type App struct {
	cfg      *config.Config
	cache    *ics.ICSCache
	browser  browser.AuthBrowser
	provider *auth.ADFSProvider
	loc      *time.Location
	sched    *scheduler.Scheduler
	srv      *server.Server
	mu       sync.Mutex
	fetching bool
	done     chan struct{}
}

func New(cfg *config.Config, cache *ics.ICSCache, browser browser.AuthBrowser, provider *auth.ADFSProvider, loc *time.Location) *App {
	return &App{
		cfg:      cfg,
		cache:    cache,
		browser:  browser,
		provider: provider,
		loc:      loc,
		done:     make(chan struct{}),
	}
}

func (a *App) Run() error {
	sched, err := scheduler.New(a.cfg.TZ)
	if err != nil {
		return err
	}
	a.sched = sched
	sched.Start()

	if err := sched.AddJob(a.cfg.FetchCron, func() { a.Fetch() }); err != nil {
		log.Printf("scheduler: %v", err)
	}

	go func() { a.Fetch() }()

	a.srv = server.NewServer(a.cfg.ServerPort, a.cache, a.cfg.TZ, a.cfg.ICSRefreshInterval)
	go func() { _ = a.srv.Start() }()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("shutting down...")

	a.Shutdown()

	return nil
}

func (a *App) Fetch() {
	a.mu.Lock()
	if a.fetching {
		a.mu.Unlock()
		return
	}
	a.fetching = true
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.fetching = false
		select {
		case a.done <- struct{}{}:
		default:
		}
		a.mu.Unlock()
	}()

	runFetch(a.cfg, a.provider, a.cache, a.loc)
}

func (a *App) Shutdown() {
	a.mu.Lock()
	if a.fetching {
		a.mu.Unlock()
		select {
		case <-a.done:
		default:
			timeout := make(chan struct{})
			go func() {
				<-a.done
				close(timeout)
			}()
			select {
			case <-timeout:
			case <-time.After(30 * time.Second):
			}
		}
	} else {
		a.mu.Unlock()
	}

	if a.sched != nil {
		a.sched.Stop()
	}

	if a.browser != nil {
		a.browser.Close()
	}

	if err := saveAll(a.cache, a.cfg.ICSStoragePath, a.cfg.ICSOnlinePath, a.cfg.ICSCampusPath, a.cfg.XsiteEventsPath, a.cfg.XsiteDropboxPath, a.cfg.TZ, a.cfg.ICSRefreshInterval); err != nil {
		log.Printf("save on shutdown failed: %v", err)
	}
}
