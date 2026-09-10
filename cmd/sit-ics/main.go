package main

import (
	"log"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/app"
	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
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

	cache := calendar.NewICSCache()
	if err := cache.LoadFromFile(cfg.ICSStoragePath, loc); err != nil {
		log.Printf("ics: failed to load from disk: %v", err)
	}

	authBrowser := createBrowser(cfg)
	provider := auth.NewADFSProvider(authBrowser)

	a := app.New(cfg, cache, authBrowser, provider, loc)

	if err := a.Run(); err != nil {
		log.Fatalf("app: %v", err)
	}
}

func createBrowser(cfg *config.Config) browser.AuthBrowser {
	switch cfg.BrowserMode {
	case config.BrowserSystem, config.BrowserAuto, config.BrowserRod:
		authBrowser, err := browser.NewLocalBrowser(browser.BrowserConfig{
			Executable:        cfg.BrowserExecutable,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       app.FetchTimeout,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
		return authBrowser
	case config.BrowserRemote:
		authBrowser, err := browser.NewRemoteBrowser(browser.BrowserConfig{
			RemoteHost:        cfg.BrowserRemoteHost,
			RemotePort:        cfg.BrowserRemotePort,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       app.FetchTimeout,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
		return authBrowser
	default:
		log.Fatalf("unsupported browser mode: %s", cfg.BrowserMode)
		return nil
	}
}
