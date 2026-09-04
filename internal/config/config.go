package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/spf13/pflag"
)

type Config struct {
	Username         string        `env:"USERNAME" envDefault:""`
	Password         string        `env:"PASSWORD" envDefault:""`
	TOTPSecret       string        `env:"TOTP_SECRET" envDefault:""`
	StartDate        time.Time     `env:"START_DATE" envDefault:""`
	EndDate          time.Time     `env:"END_DATE" envDefault:""`
	TZ               string        `env:"TZ" envDefault:"Asia/Singapore"`
	FetchCron        string        `env:"FETCH_CRON" envDefault:"0 1 * * *"`
	ServerPort       int           `env:"SERVER_PORT" envDefault:"8080"`
	ICSStoragePath   string        `env:"ICS_STORAGE_PATH" envDefault:"./timetable.ics"`
	BrowserMode      BrowserMode   `env:"BROWSER_MODE" envDefault:"auto"`
	BrowserExecutable  string `env:"BROWSER_EXECUTABLE" envDefault:""`
	BrowserRemoteHost  string `env:"BROWSER_REMOTE_HOST" envDefault:""`
	BrowserRemotePort  int    `env:"BROWSER_REMOTE_PORT" envDefault:"9222"`
	BrowserHeadless    bool   `env:"BROWSER_HEADLESS" envDefault:"true"`
	BrowserDebug                      bool    `env:"BROWSER_DEBUG" envDefault:"false"`
	ProxyURL                          string  `env:"PROXY_URL" envDefault:""`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env vars: %w", err)
	}

	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	username := fs.String("username", "", "ADFS username")
	password := fs.String("password", "", "ADFS password")
	totpSecret := fs.String("totp-secret", "", "TOTP secret")
	startDate := fs.String("start-date", "", "Start date (YYYY-MM-DD)")
	endDate := fs.String("end-date", "", "End date (YYYY-MM-DD)")
	tz := fs.String("tz", "", "Timezone (IANA name)")
	fetchCron := fs.String("fetch-cron", "", "Cron schedule for fetches")
	serverPort := fs.Int("server-port", 0, "HTTP server port")
	icsStoragePath := fs.String("ics-storage-path", "", "Path to ICS file")
	browserMode := fs.String("browser-mode", "", "Browser mode (auto/system/rod/remote)")
	browserExecutable := fs.String("browser-executable", "", "Browser executable path")
	browserRemoteHost := fs.String("browser-remote-host", "", "Remote browser host (IP or FQDN)")
	browserRemotePort := fs.Int("browser-remote-port", 0, "Remote browser port")
	browserHeadless := fs.Bool("browser-headless", false, "Run browser in headless mode")
	browserDebug := fs.Bool("browser-debug", false, "Enable debug logging for browser actions")
	proxyURL := fs.String("proxy-url", "", "SOCKS5 proxy URL (e.g. socks5://localhost:1080)")

	fs.Parse(os.Args[1:])

	if fs.Lookup("username").Changed {
		cfg.Username = *username
	}
	if fs.Lookup("password").Changed {
		cfg.Password = *password
	}
	if fs.Lookup("totp-secret").Changed {
		cfg.TOTPSecret = *totpSecret
	}
	if fs.Lookup("start-date").Changed && *startDate != "" {
		t, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			return nil, fmt.Errorf("parse start-date %q: %w", *startDate, err)
		}
		cfg.StartDate = t
	}
	if fs.Lookup("end-date").Changed && *endDate != "" {
		t, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			return nil, fmt.Errorf("parse end-date %q: %w", *endDate, err)
		}
		cfg.EndDate = t
	}
	if fs.Lookup("tz").Changed {
		cfg.TZ = *tz
	}
	if fs.Lookup("fetch-cron").Changed {
		cfg.FetchCron = *fetchCron
	}
	if fs.Lookup("server-port").Changed {
		cfg.ServerPort = *serverPort
	}
	if fs.Lookup("ics-storage-path").Changed {
		cfg.ICSStoragePath = *icsStoragePath
	}
	if fs.Lookup("browser-mode").Changed {
		cfg.BrowserMode = BrowserMode(*browserMode)
	}
	if fs.Lookup("browser-executable").Changed {
		cfg.BrowserExecutable = *browserExecutable
	}
	if fs.Lookup("browser-remote-host").Changed {
		cfg.BrowserRemoteHost = *browserRemoteHost
	}
	if fs.Lookup("browser-remote-port").Changed {
		cfg.BrowserRemotePort = *browserRemotePort
	}
	if fs.Lookup("browser-headless").Changed {
		cfg.BrowserHeadless = *browserHeadless
	}
	if fs.Lookup("browser-debug").Changed {
		cfg.BrowserDebug = *browserDebug
	}
	if fs.Lookup("proxy-url").Changed {
		cfg.ProxyURL = *proxyURL
	}

	return &cfg, nil
}
