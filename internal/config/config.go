package config

import (
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/credentialstore"
)

type Config struct {
	Username                          string        `env:"USERNAME" envDefault:""`
	Password                          string        `env:"PASSWORD" envDefault:""`
	TOTPSecret                        string        `env:"TOTP_SECRET" envDefault:""`
	StartDate                         time.Time     `env:"START_DATE" envDefault:""`
	EndDate                           time.Time     `env:"END_DATE" envDefault:""`
	TZ                                string        `env:"TZ" envDefault:"Asia/Singapore"`
	FetchCron                         string        `env:"FETCH_CRON" envDefault:"0 1 * * *"`
	ServerPort                        int           `env:"SERVER_PORT" envDefault:"8080"`
	ServerAddr                        string        `env:"SERVER_ADDR" envDefault:""`
	ICSStoragePath                    string        `env:"ICS_STORAGE_PATH" envDefault:"./timetable.ics"`
	ICSOnlinePath                     string        `env:"ICS_ONLINE_PATH" envDefault:"./timetable-online.ics"`
	ICSCampusPath                     string        `env:"ICS_CAMPUS_PATH" envDefault:"./timetable-campus.ics"`
	BrightSpaceEventsPath             string        `env:"BRIGHTSPACE_EVENTS_PATH" envDefault:"./brightspace-events.ics"`
	BrightSpaceDropboxPath            string        `env:"BRIGHTSPACE_DROPBOX_PATH" envDefault:"./brightspace-dropbox.ics"`
	BrowserMode                       BrowserMode   `env:"BROWSER_MODE" envDefault:"auto"`
	BrowserExecutable                 string        `env:"BROWSER_EXECUTABLE" envDefault:""`
	BrowserRemoteHost                 string        `env:"BROWSER_REMOTE_HOST" envDefault:""`
	BrowserRemotePort                 int           `env:"BROWSER_REMOTE_PORT" envDefault:"9222"`
	BrowserHeadless                   bool          `env:"BROWSER_HEADLESS" envDefault:"true"`
	BrowserDebug                      bool          `env:"BROWSER_DEBUG" envDefault:"false"`
	ProxyURL                          string        `env:"PROXY_URL" envDefault:""`
	ICSRefreshInterval                time.Duration `env:"ICS_REFRESH_INTERVAL" envDefault:"1h"`
	BrightSpaceEnabled                bool          `env:"BRIGHTSPACE_ENABLED" envDefault:"false"`
	BrightSpaceBaseURL                string        `env:"BRIGHTSPACE_BASE_URL" envDefault:"https://xsite.singaporetech.edu.sg"`
	BrightSpaceAPIKey                 string        `env:"BRIGHTSPACE_API_KEY" envDefault:""`
	BrightSpaceAPISecret              string        `env:"BRIGHTSPACE_API_SECRET" envDefault:""`
	BrightSpaceCourseNameBlocklist    string        `env:"BRIGHTSPACE_COURSE_NAME_BLOCKLIST" envDefault:""`
	BrightSpaceCourseIDBlocklist      string        `env:"BRIGHTSPACE_COURSE_ID_BLOCKLIST" envDefault:""`
	BrightSpaceEventTitleBlocklist    string        `env:"BRIGHTSPACE_EVENT_TITLE_BLOCKLIST" envDefault:""`
	BrightSpaceEventLocationBlocklist string        `env:"BRIGHTSPACE_EVENT_LOCATION_BLOCKLIST" envDefault:""`
	BrightSpaceQuizzesPath            string        `env:"BRIGHTSPACE_QUIZZES_PATH" envDefault:"./brightspace-quizzes.ics"`
	BrightSpaceQuizTitleBlocklist     string        `env:"BRIGHTSPACE_QUIZ_TITLE_BLOCKLIST" envDefault:""`
	BrightSpaceQuizzesAlerts          string        `env:"BRIGHTSPACE_QUIZZES_ALERTS" envDefault:""`
	TimetableAlerts                   string        `env:"TIMETABLE_ALERTS" envDefault:""`
	TimetableOnlineAlerts             string        `env:"TIMETABLE_ONLINE_ALERTS" envDefault:""`
	ICSCampusAlerts                   string        `env:"ICS_CAMPUS_ALERTS" envDefault:""`
	BrightSpaceEventsAlerts           string        `env:"BRIGHTSPACE_EVENTS_ALERTS" envDefault:""`
	BrightSpaceDropboxAlerts          string        `env:"BRIGHTSPACE_DROPBOX_ALERTS" envDefault:""`

	credStore credentialstore.Store
}

func Load() (*Config, error) {
	cfg, err := loadEnv()
	if err != nil {
		return nil, err
	}

	cfg.credStore = credentialstore.NewStore()

	if err := applyFlags(cfg); err != nil {
		return nil, err
	}

	if err := cfg.loadCredentialsFromStore(); err != nil {
		return nil, err
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) loadCredentialsFromStore() error {
	if c.credStore == nil {
		return nil
	}

	if c.Username == "" {
		if username, err := c.credStore.GetUsername(); err == nil && username != "" {
			c.Username = username
		}
	}
	if c.Password == "" {
		if password, err := c.credStore.GetPassword(); err == nil && password != "" {
			c.Password = password
		}
	}
	if c.TOTPSecret == "" {
		if totp, err := c.credStore.GetTOTPSecret(); err == nil && totp != "" {
			c.TOTPSecret = totp
		}
	}

	return nil
}
