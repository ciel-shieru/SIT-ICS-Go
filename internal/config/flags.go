package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/spf13/pflag"
)

func loadEnv() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env vars: %w", err)
	}
	return &cfg, nil
}

func applyFlags(cfg *Config) error {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	username := fs.String("username", "", "ADFS username")
	password := fs.String("password", "", "ADFS password")
	totpSecret := fs.String("totp-secret", "", "TOTP secret")
	startDate := fs.String("start-date", "", "Start date (YYYY-MM-DD)")
	endDate := fs.String("end-date", "", "End date (YYYY-MM-DD)")
	tz := fs.String("tz", "", "Timezone (IANA name)")
	fetchCron := fs.String("fetch-cron", "", "Cron schedule for fetches")
	serverPort := fs.Int("server-port", 0, "HTTP server port")
	icsStoragePath := fs.String("ics-storage-path", "", "Path to main ICS file")
	icsOnlinePath := fs.String("ics-online-path", "", "Path to online-only ICS file")
	icsCampusPath := fs.String("ics-campus-path", "", "Path to campus-only ICS file")
	browserMode := fs.String("browser-mode", "", "Browser mode (auto/system/rod/remote)")
	browserExecutable := fs.String("browser-executable", "", "Browser executable path")
	browserRemoteHost := fs.String("browser-remote-host", "", "Remote browser host (IP or FQDN)")
	browserRemotePort := fs.Int("browser-remote-port", 0, "Remote browser port")
	browserHeadless := fs.Bool("browser-headless", false, "Run browser in headless mode")
	browserDebug := fs.Bool("browser-debug", false, "Enable debug logging for browser actions")
	proxyURL := fs.String("proxy-url", "", "SOCKS5 proxy URL (e.g. socks5://localhost:1080)")
	icsRefreshInterval := fs.Duration("ics-refresh-interval", 0, "ICS refresh interval (e.g. 1h, 30m)")
	xsiteEventsPath := fs.String("xsite-events-path", "", "Path to xsite events ICS file")
	xsiteDropboxPath := fs.String("xsite-dropbox-path", "", "Path to xsite dropbox ICS file")
	brightspaceEnabled := fs.Bool("brightspace-enabled", false, "Enable BrightSpace D2L extraction")
	brightspaceBaseURL := fs.String("brightspace-base-url", "", "BrightSpace D2L base URL")
	brightspaceAPIKey := fs.String("brightspace-api-key", "", "BrightSpace D2L API key")
	brightspaceAPISecret := fs.String("brightspace-api-secret", "", "BrightSpace D2L API secret")
	brightspaceCourseNameBlocklist := fs.String("brightspace-course-name-blocklist", "", "Comma-separated course name patterns to block")
	brightspaceCourseIDBlocklist := fs.String("brightspace-course-id-blocklist", "", "Comma-separated course OrgUnitIds to block")
	brightspaceEventTitleBlocklist := fs.String("brightspace-event-title-blocklist", "", "Comma-separated event title patterns to block")
	brightspaceEventLocationBlocklist := fs.String("brightspace-event-location-blocklist", "", "Comma-separated event location patterns to block")

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
			return fmt.Errorf("parse start-date %q: %w", *startDate, err)
		}
		cfg.StartDate = t
	}
	if fs.Lookup("end-date").Changed && *endDate != "" {
		t, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			return fmt.Errorf("parse end-date %q: %w", *endDate, err)
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
	if fs.Lookup("ics-online-path").Changed {
		cfg.ICSOnlinePath = *icsOnlinePath
	}
	if fs.Lookup("ics-campus-path").Changed {
		cfg.ICSCampusPath = *icsCampusPath
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
	if fs.Lookup("ics-refresh-interval").Changed {
		cfg.ICSRefreshInterval = *icsRefreshInterval
	}
	if fs.Lookup("xsite-events-path").Changed {
		cfg.XsiteEventsPath = *xsiteEventsPath
	}
	if fs.Lookup("xsite-dropbox-path").Changed {
		cfg.XsiteDropboxPath = *xsiteDropboxPath
	}
	if fs.Lookup("brightspace-enabled").Changed {
		cfg.BrightSpaceEnabled = *brightspaceEnabled
	}
	if fs.Lookup("brightspace-base-url").Changed {
		cfg.BrightSpaceBaseURL = *brightspaceBaseURL
	}
	if fs.Lookup("brightspace-api-key").Changed {
		cfg.BrightSpaceAPIKey = *brightspaceAPIKey
	}
	if fs.Lookup("brightspace-api-secret").Changed {
		cfg.BrightSpaceAPISecret = *brightspaceAPISecret
	}
	if fs.Lookup("brightspace-course-name-blocklist").Changed {
		cfg.BrightSpaceCourseNameBlocklist = *brightspaceCourseNameBlocklist
	}
	if fs.Lookup("brightspace-course-id-blocklist").Changed {
		cfg.BrightSpaceCourseIDBlocklist = *brightspaceCourseIDBlocklist
	}
	if fs.Lookup("brightspace-event-title-blocklist").Changed {
		cfg.BrightSpaceEventTitleBlocklist = *brightspaceEventTitleBlocklist
	}
	if fs.Lookup("brightspace-event-location-blocklist").Changed {
		cfg.BrightSpaceEventLocationBlocklist = *brightspaceEventLocationBlocklist
	}

	return nil
}
