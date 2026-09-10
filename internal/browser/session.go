package browser

import (
	"errors"
	"time"
)

type AuthRequest struct {
	URL        string
	Username   string
	Password   string
	TOTPSecret string
}

type Cookie struct {
	Name   string
	Value  string
	Domain string
	Path   string
	Expiry int64
}

type BrowserConfig struct {
	Mode              BrowserMode
	Executable        string
	RemoteHost        string
	RemotePort        int
	Headless          bool
	Incognito         bool
	ProxyURL          string
	Debug             bool
	ConnectTimeout    time.Duration
	NavigationTimeout time.Duration
	AuthTimeout       time.Duration
}

type BrowserMode string

const (
	BrowserModeAuto   BrowserMode = "auto"
	BrowserModeSystem BrowserMode = "system"
	BrowserModeRod    BrowserMode = "rod"
	BrowserModeRemote BrowserMode = "remote"
)

var (
	ErrBrowserUnavailable    = errors.New("browser unavailable")
	ErrBrowserLaunch         = errors.New("browser launch failed")
	ErrBrowserConnect        = errors.New("browser connection failed")
	ErrNavigation            = errors.New("navigation failed")
	ErrAuthentication        = errors.New("authentication failed")
	ErrAuthenticationTimeout = errors.New("authentication timed out")
	ErrCredentialExtraction  = errors.New("credential extraction failed")
)
