package main

import (
	"log"
	"time"
	_ "time/tzdata"

	"github.com/ciel-shieru/sit-ics-go/internal/app"
	"github.com/ciel-shieru/sit-ics-go/internal/auth"
	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/calendar"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/credentialstore"
	"github.com/ciel-shieru/sit-ics-go/internal/credprompt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := credprompt.PromptIfNeeded(cfg, cfg.OverrideCredentials); err != nil {
		log.Fatalf("credential prompt: %v", err)
	}

	defer credentialstore.NewStore().Close()

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("invalid timezone %q: %v", cfg.TZ, err)
	}

	cache := calendar.NewICSCache()
	if err := cache.LoadFromFile(cfg.ICSStoragePath, loc); err != nil {
		log.Printf("ics: failed to load from disk: %v", err)
	}

	var authBrowser browser.AuthBrowser
	if cfg.BrowserMode == config.BrowserRemote {
		var err error
		authBrowser, err = browser.NewRemoteBrowser(browser.BrowserConfig{
			Mode:              cfg.BrowserMode,
			RemoteHost:        cfg.BrowserRemoteHost,
			RemotePort:        cfg.BrowserRemotePort,
			Headless:          cfg.BrowserHeadless,
			Incognito:         true,
			ProxyURL:          cfg.ProxyURL,
			Debug:             cfg.BrowserDebug,
			ConnectTimeout:    10 * time.Second,
			NavigationTimeout: 30 * time.Second,
			AuthTimeout:       app.FetchTimeout,
			RetryInterval:     cfg.BrowserRetryInterval,
			MaxRetries:        cfg.BrowserMaxRetries,
		})
		if err != nil {
			log.Fatalf("browser: %v", err)
		}
	}
	provider := auth.NewADFSProvider(authBrowser)

	a := app.New(cfg, cache, provider, authBrowser, loc)

	if err := a.Run(); err != nil {
		log.Fatalf("app: %v", err)
	}
}
