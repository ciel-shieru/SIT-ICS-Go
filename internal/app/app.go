package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/scheduler"
	"github.com/ciel-shieru/sit-ics-go/internal/server"
)

const (
	FetchTimeout = 5 * time.Minute

	TimetableFetchTimeout = 5 * time.Minute

	BrightSpaceFetchTimeout = 5 * time.Minute
)

type App struct {
	cfg           *config.Config
	cache         *calendar.ICSCache
	provider      *auth.ADFSProvider
	loc           *time.Location
	sched         *scheduler.Scheduler
	srv           *server.Server
	remoteBrowser browser.AuthBrowser
	mu            sync.Mutex
	fetching      bool
	refreshPending bool
	refreshScheduled bool
	shuttingDown bool
	lastFetchJobTime   time.Time
	lastStatusIN4SIT   string
	lastStatusXsite    string
	done               chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

func New(cfg *config.Config, cache *calendar.ICSCache, provider *auth.ADFSProvider, remoteBrowser browser.AuthBrowser, loc *time.Location) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		cfg:           cfg,
		cache:         cache,
		provider:      provider,
		loc:           loc,
		remoteBrowser: remoteBrowser,
		lastStatusIN4SIT: "unknown",
		lastStatusXsite:  "unknown",
		done:             make(chan struct{}),
		ctx:              ctx,
		cancel:        cancel,
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

	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts := calendar.AlertsFromConfig(
		a.cfg.TimetableAlerts,
		a.cfg.TimetableOnlineAlerts,
		a.cfg.TimetableCampusAlerts,
		a.cfg.XsiteEventsAlerts,
		a.cfg.XsiteDropboxAlerts,
		a.cfg.XsiteQuizzesAlerts,
	)
	a.srv = server.NewServer(a.cfg.ServerPort, a.cfg.ServerAddr, a.cache, a.cfg.TZ, a.cfg.ICSRefreshInterval, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts, a.cfg.ServerTrustedProxies, a.cfg.ServerDisableCaching, a.LastFetchJobTime, a.FetchStatuses, a.RequestRefresh)
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
	if a.shuttingDown || a.fetching {
		a.mu.Unlock()
		return
	}
	a.fetching = true
	a.lastFetchJobTime = time.Now().UTC()
	a.mu.Unlock()

	a.runFetchJob()
}

func (a *App) runFetchJob() {
	defer func() {
		a.mu.Lock()
		a.fetching = false
		pending := a.refreshPending && !a.shuttingDown
		a.refreshPending = false
		select {
		case a.done <- struct{}{}:
		default:
		}
		a.mu.Unlock()
		if pending {
			a.RequestRefresh()
		}
	}()

	b, err := a.spawnBrowser()
	if err != nil {
		log.Printf("scheduler: failed to spawn browser: %v", err)
		return
	}

	if b != nil {
		defer b.Close()
		old := a.provider.SetBrowser(b)
		defer a.provider.SetBrowser(old)
	}

	runFetchWithStatus(a.ctx, a.cfg, a.provider, a.cache, a.loc, a.setLastStatusIN4SIT, a.setLastStatusXsite)
}

// FetchStatuses returns current overall-job and last source-fetch statuses.
// Source statuses persist until the next corresponding fetch starts.
func (a *App) FetchStatuses() (jobStatus, lastStatusIN4SIT, lastStatusXsite string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	jobStatus = "finished"
	if a.fetching {
		jobStatus = "running"
	}
	lastStatusIN4SIT = a.lastStatusIN4SIT
	if lastStatusIN4SIT == "" {
		lastStatusIN4SIT = "unknown"
	}
	lastStatusXsite = a.lastStatusXsite
	if lastStatusXsite == "" {
		lastStatusXsite = "unknown"
	}
	return jobStatus, lastStatusIN4SIT, lastStatusXsite
}

func (a *App) setLastStatusIN4SIT(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastStatusIN4SIT = status
}

func (a *App) setLastStatusXsite(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastStatusXsite = status
}

// LastFetchJobTime returns the start time of the most recent fetch job.
// It reads only in-memory state.
func (a *App) LastFetchJobTime() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastFetchJobTime
}

// RequestRefresh queues one immediate fetch. Requests during a running job are
// coalesced into a single follow-up job instead of being silently discarded.
func (a *App) RequestRefresh() {
	a.mu.Lock()
	if a.shuttingDown {
		a.mu.Unlock()
		return
	}
	if a.fetching {
		a.refreshPending = true
		a.mu.Unlock()
		return
	}
	if a.refreshScheduled {
		a.mu.Unlock()
		return
	}
	a.refreshScheduled = true
	a.mu.Unlock()

	go func() {
		a.mu.Lock()
		a.refreshScheduled = false
		if a.shuttingDown {
			a.mu.Unlock()
			return
		}
		if a.fetching {
			a.refreshPending = true
			a.mu.Unlock()
			return
		}
		a.fetching = true
		a.lastFetchJobTime = time.Now().UTC()
		a.mu.Unlock()
		a.runFetchJob()
	}()
}

func (a *App) spawnBrowser() (browser.AuthBrowser, error) {
	if a.cfg.BrowserMode == config.BrowserRemote {
		return nil, nil
	}

	b, err := browser.NewLocalBrowser(browser.BrowserConfig{
		Mode:              a.cfg.BrowserMode,
		Executable:        a.cfg.BrowserExecutable,
		Headless:          a.cfg.BrowserHeadless,
		Incognito:         true,
		ProxyURL:          a.cfg.ProxyURL,
		Debug:             a.cfg.BrowserDebug,
		ConnectTimeout:    10 * time.Second,
		NavigationTimeout: 30 * time.Second,
		AuthTimeout:       FetchTimeout,
		RetryInterval:     a.cfg.BrowserRetryInterval,
		MaxRetries:        a.cfg.BrowserMaxRetries,
	})
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (a *App) Shutdown() {
	a.mu.Lock()
	a.shuttingDown = true
	if a.fetching {
		a.cancel()
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

	if a.remoteBrowser != nil {
		a.remoteBrowser.Close()
	}

	mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts := calendar.AlertsFromConfig(
		a.cfg.TimetableAlerts,
		a.cfg.TimetableOnlineAlerts,
		a.cfg.TimetableCampusAlerts,
		a.cfg.XsiteEventsAlerts,
		a.cfg.XsiteDropboxAlerts,
		a.cfg.XsiteQuizzesAlerts,
	)
	if err := saveAllOutputs(a.cache, a.cfg.ICSStoragePath, a.cfg.ICSOnlinePath, a.cfg.ICSCampusPath, a.cfg.XsiteEventsPath, a.cfg.XsiteDropboxPath, a.cfg.XsiteQuizzesPath, a.cfg.XsitePath, a.cfg.TZ, a.cfg.ICSRefreshInterval, mainAlerts, onlineAlerts, campusAlerts, bsEventsAlerts, bsDropboxAlerts, bsQuizzesAlerts); err != nil {
		log.Printf("save on shutdown failed: %v", err)
	}
}
