package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/brightspace"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

type AuthRequest struct {
	Username      string
	Password      string
	TOTPSecret    string
	PeopleSoftURL string
}

// ADFSProvider wraps AuthBrowser and provides a higher-level interface
// for authentication and timetable fetching.
type ADFSProvider struct {
	browser browser.AuthBrowser
}

// NewADFSProvider creates a new ADFSProvider that uses the given AuthBrowser
// for authentication and data fetching.
func NewADFSProvider(b browser.AuthBrowser) *ADFSProvider {
	return &ADFSProvider{
		browser: b,
	}
}

// Authenticate performs ADFS authentication using the wrapped browser
// and returns the redirect URL after successful authentication.
func (a *ADFSProvider) Authenticate(ctx context.Context, req AuthRequest) (browser.AuthResult, error) {
	browserReq := browser.AuthRequest{
		URL:        "https://in4sit.singaporetech.edu.sg",
		Username:   req.Username,
		Password:   req.Password,
		TOTPSecret: req.TOTPSecret,
	}

	authResult, err := a.browser.Authenticate(ctx, browserReq)
	if err != nil {
		return browser.AuthResult{}, fmt.Errorf("adfs authenticate: %w", err)
	}

	return authResult, nil
}

// FetchTimetable fetches and parses the timetable from PeopleSoft.
func (a *ADFSProvider) FetchTimetable(ctx context.Context, weekDate string, loc *time.Location) ([]peoplesoft.Entry, error) {
	html, err := a.browser.FetchTimetable(ctx, weekDate)
	if err != nil {
		return nil, fmt.Errorf("browser fetch timetable: %w", err)
	}
	year := peoplesoft.ExtractYear(weekDate)
	return peoplesoft.ParseTimetableHTML(html, year, loc)
}

// FetchBrightSpace fetches BrightSpace entries using the existing authenticated session.
func (a *ADFSProvider) FetchBrightSpace(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
	entries, err := a.browser.FetchBrightSpace(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("browser fetch brightspace: %w", err)
	}
	return entries, nil
}

// FetchBrightSpaceQuizzes fetches BrightSpace quiz entries using the existing authenticated session.
func (a *ADFSProvider) FetchBrightSpaceQuizzes(ctx context.Context, baseURL string) ([]brightspace.BrightSpaceStringEntry, error) {
	entries, err := a.browser.FetchBrightSpaceQuizzes(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("browser fetch brightspace quizzes: %w", err)
	}
	return entries, nil
}
