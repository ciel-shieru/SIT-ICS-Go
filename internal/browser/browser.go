package browser

import (
	"context"
	"time"
)

type AuthResult struct {
	Cookies     []Cookie
	RedirectURL string
	Token       string
}

type BrightSpaceEntry struct {
	Title       string
	OrgUnitId   string
	OrgUnitName string
	OrgUnitCode string
	Location    string
	Description string
	DTStart     string
	DTEnd       string
	IsAllDay    bool
	Source      string
}

type AuthBrowser interface {
	Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error)
	FetchTimetable(ctx context.Context, weekDate string) (string, error)
	FetchBrightSpace(ctx context.Context, baseURL string) ([]BrightSpaceEntry, error)
	Close()
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
	"https://xsite.singaporetech.edu.sg",
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
	AuthenticateFunc     func(ctx context.Context, req AuthRequest) (AuthResult, error)
	FetchTimetableFunc   func(ctx context.Context, weekDate string) (string, error)
	FetchBrightSpaceFunc func(ctx context.Context, baseURL string) ([]BrightSpaceEntry, error)
}

func (m *MockAuthBrowser) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	if m.AuthenticateFunc != nil {
		return m.AuthenticateFunc(ctx, req)
	}
	return AuthResult{}, nil
}

func (m *MockAuthBrowser) FetchTimetable(ctx context.Context, weekDate string) (string, error) {
	if m.FetchTimetableFunc != nil {
		return m.FetchTimetableFunc(ctx, weekDate)
	}
	return "", nil
}

func (m *MockAuthBrowser) FetchBrightSpace(ctx context.Context, baseURL string) ([]BrightSpaceEntry, error) {
	if m.FetchBrightSpaceFunc != nil {
		return m.FetchBrightSpaceFunc(ctx, baseURL)
	}
	return nil, nil
}

func (m *MockAuthBrowser) Close() {}
