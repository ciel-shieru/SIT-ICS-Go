package browser

import (
	"context"
	"time"
)

type BrowserMode string

const (
	BrowserAuto   BrowserMode = "auto"
	BrowserSystem BrowserMode = "system"
	BrowserRod    BrowserMode = "rod"
	BrowserRemote BrowserMode = "remote"
)

type Cookie struct {
	Name   string
	Value  string
	Domain string
	Path   string
}

type AuthRequest struct {
	URL        string
	Username   string
	Password   string
	TOTPSecret string
}

type AuthResult struct {
	Cookies     []Cookie
	RedirectURL string
	Token       string
}

type AuthBrowser interface {
	Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error)
}

type BrowserConfig struct {
	Mode              BrowserMode
	Executable        string
	ControlURL        string
	Headless          bool
	Incognito         bool
	ProxyURL          string
	ConnectTimeout    time.Duration
	NavigationTimeout time.Duration
	AuthTimeout       time.Duration
}

func DefaultBrowserConfig(mode BrowserMode) BrowserConfig {
	return BrowserConfig{
		Mode:              mode,
		Headless:          true,
		Incognito:         true,
		ConnectTimeout:    10 * time.Second,
		NavigationTimeout: 30 * time.Second,
		AuthTimeout:       5 * time.Minute,
	}
}

var allowedOrigins = []string{
	"https://in4sit.singaporetech.edu.sg",
	"https://fs.singaporetech.edu.sg",
}

func isAllowedOrigin(url string) bool {
	for _, origin := range allowedOrigins {
		if url == origin || len(url) >= len(origin) && url[:len(origin)] == origin && (url[len(origin)] == '/' || url[len(origin)] == ':') {
			return true
		}
	}
	return false
}

type MockAuthBrowser struct {
	AuthenticateFunc func(ctx context.Context, req AuthRequest) (AuthResult, error)
}

func (m *MockAuthBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(ctx, req)
	}
	return AuthResult{}, nil
}
