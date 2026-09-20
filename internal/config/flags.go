package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"
)

func applyFlags(cfg *Config) error {
	fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
	startDate := fs.String("start-date", "", "Start date (YYYY-MM-DD) (env: START_DATE)")
	endDate := fs.String("end-date", "", "End date (YYYY-MM-DD) (env: END_DATE)")
	tz := fs.String("tz", "", "Timezone (IANA name) (env: TZ)")
	fetchCron := fs.String("fetch-cron", "", "Cron schedule for fetches (env: FETCH_CRON)")
	serverPort := fs.Int("server-port", 0, "HTTP server port (env: SERVER_PORT)")
	serverAddr := fs.String("server-addr", "", "HTTP server bind address (e.g. 127.0.0.1, 0.0.0.0) (env: SERVER_ADDR)")
	icsStoragePath := fs.String("ics-storage-path", "", "Path to main ICS file (env: ICS_STORAGE_PATH)")
	icsOnlinePath := fs.String("ics-online-path", "", "Path to online-only ICS file (env: ICS_ONLINE_PATH)")
	icsCampusPath := fs.String("ics-campus-path", "", "Path to campus-only ICS file (env: ICS_CAMPUS_PATH)")
	browserMode                        := fs.String("browser-mode", "", "Browser mode (auto/system/rod/remote) (env: BROWSER_MODE)")
	browserExecutable                  := fs.String("browser-executable", "", "Browser executable path (env: BROWSER_EXECUTABLE)")
	browserRemoteHost                  := fs.String("browser-remote-host", "", "Remote browser host (IP or FQDN) (env: BROWSER_REMOTE_HOST)")
	browserRemotePort                  := fs.Int("browser-remote-port", 0, "Remote browser port (env: BROWSER_REMOTE_PORT)")
	browserHeadless                    := fs.Bool("browser-headless", false, "Run browser in headless mode (env: BROWSER_HEADLESS)")
	browserDebug                       := fs.Bool("browser-debug", false, "Enable debug logging for browser actions (env: BROWSER_DEBUG)")
	proxyURL                           := fs.String("proxy-url", "", "SOCKS5 proxy URL (e.g. socks5://localhost:1080) (env: PROXY_URL)")
	icsRefreshInterval                 := fs.Duration("ics-refresh-interval", 0, "ICS refresh interval (e.g. 1h, 30m) (env: ICS_REFRESH_INTERVAL)")
	xsiteEventsPath                    := fs.String("xsite-events-path", "", "Path to xsite events ICS file (env: XSITE_EVENTS_PATH)")
	xsiteDropboxPath                   := fs.String("xsite-dropbox-path", "", "Path to xsite dropbox ICS file (env: XSITE_DROPBOX_PATH)")
	xsiteEnabled                       := fs.Bool("xsite-enabled", false, "Enable xSite D2L extraction (env: XSITE_ENABLED)")
	xsitePath                          := fs.String("xsite-path", "", "Path to xsite combo ICS file (env: XSITE_PATH)")
	xsiteCourseNameBlocklist           := fs.String("xsite-course-name-blocklist", "", "Comma-separated course name patterns to block (env: XSITE_COURSE_NAME_BLOCKLIST)")
	xsiteCourseIDBlocklist             := fs.String("xsite-course-id-blocklist", "", "Comma-separated course OrgUnitIds to block (env: XSITE_COURSE_ID_BLOCKLIST)")
	xsiteEventTitleBlocklist           := fs.String("xsite-event-title-blocklist", "", "Comma-separated event title patterns to block (env: XSITE_EVENT_TITLE_BLOCKLIST)")
	xsiteEventLocationBlocklist        := fs.String("xsite-event-location-blocklist", "", "Comma-separated event location patterns to block (env: XSITE_EVENT_LOCATION_BLOCKLIST)")
	timetableOnlineAlerts              := fs.String("timetable-online-alerts", "", "Comma-separated ICS duration strings for timetable online VALARM (e.g. -P2D,-P1D) (env: TIMETABLE_ONLINE_ALERTS)")
	timetableCampusAlerts              := fs.String("timetable-campus-alerts", "", "Comma-separated ICS duration strings for timetable in-campus VALARM (env: TIMETABLE_CAMPUS_ALERTS)")
	xsiteEventsAlerts                  := fs.String("xsite-events-alerts", "", "Comma-separated ICS duration strings for xsite events VALARM (env: XSITE_EVENTS_ALERTS)")
	xsiteDropboxAlerts                 := fs.String("xsite-dropbox-alerts", "", "Comma-separated ICS duration strings for xsite dropbox VALARM (env: XSITE_DROPBOX_ALERTS)")
	xsiteQuizzesPath                   := fs.String("xsite-quizzes-path", "", "Path to xsite quizzes ICS file (env: XSITE_QUIZZES_PATH)")
	timetableAlerts                    := fs.String("timetable-alerts", "", "Comma-separated ICS duration strings for main timetable VALARM (e.g. -P2D,-P1D) (env: TIMETABLE_ALERTS)")

	fs.Parse(os.Args[1:])

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
	if fs.Lookup("server-addr").Changed {
		cfg.ServerAddr = *serverAddr
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
	if fs.Lookup("xsite-enabled").Changed {
		cfg.XsiteEnabled = *xsiteEnabled
	}
	if fs.Lookup("xsite-path").Changed {
		cfg.XsitePath = *xsitePath
	}
	if fs.Lookup("xsite-course-name-blocklist").Changed {
		cfg.XsiteCourseNameBlocklist = *xsiteCourseNameBlocklist
	}
	if fs.Lookup("xsite-course-id-blocklist").Changed {
		cfg.XsiteCourseIDBlocklist = *xsiteCourseIDBlocklist
	}
	if fs.Lookup("xsite-event-title-blocklist").Changed {
		cfg.XsiteEventTitleBlocklist = *xsiteEventTitleBlocklist
	}
	if fs.Lookup("xsite-event-location-blocklist").Changed {
		cfg.XsiteEventLocationBlocklist = *xsiteEventLocationBlocklist
	}
	if fs.Lookup("timetable-online-alerts").Changed {
		cfg.TimetableOnlineAlerts = *timetableOnlineAlerts
	}
	if fs.Lookup("timetable-campus-alerts").Changed {
		cfg.TimetableCampusAlerts = *timetableCampusAlerts
	}
	if fs.Lookup("xsite-events-alerts").Changed {
		cfg.XsiteEventsAlerts = *xsiteEventsAlerts
	}
	if fs.Lookup("xsite-dropbox-alerts").Changed {
		cfg.XsiteDropboxAlerts = *xsiteDropboxAlerts
	}
	if fs.Lookup("xsite-quizzes-path").Changed {
		cfg.XsiteQuizzesPath = *xsiteQuizzesPath
	}
	if fs.Lookup("timetable-alerts").Changed {
		cfg.TimetableAlerts = *timetableAlerts
	}

	return nil
}
