package auth

import (
	"context"
	"fmt"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

type PeoplesoftClient interface {
	FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error)
}

type AuthRequest struct {
	Username      string
	Password      string
	TOTPSecret    string
	PeopleSoftURL string
}

type AuthResult struct {
	Cookies []browser.Cookie
	Token   string
}

type ADFSProvider struct {
	browser browser.AuthBrowser
}

func NewADFSProvider(b browser.AuthBrowser) *ADFSProvider {
	return &ADFSProvider{
		browser: b,
	}
}

func (a *ADFSProvider) Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error) {
	browserReq := browser.AuthRequest{
		URL:        "https://in4sit.singaporetech.edu.sg",
		Username:   req.Username,
		Password:   req.Password,
		TOTPSecret: req.TOTPSecret,
	}

	authResult, err := a.browser.Authenticate(ctx, browserReq)
	if err != nil {
		return AuthResult{}, fmt.Errorf("adfs authenticate: %w", err)
	}

	token := extractPS_TOKEN(authResult.Cookies)

	return AuthResult{
		Cookies: authResult.Cookies,
		Token:   token,
	}, nil
}

func (a *ADFSProvider) FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
	html, err := a.browser.FetchTimetable(ctx, weekDate)
	if err != nil {
		return nil, fmt.Errorf("browser fetch timetable: %w", err)
	}
	year := peoplesoft.ExtractYear(weekDate)
	return peoplesoft.ParseTimetableHTML(html, year)
}

func extractPS_TOKEN(cookies []browser.Cookie) string {
	for _, c := range cookies {
		if c.Name == "PS_TOKEN" {
			return c.Value
		}
	}
	return ""
}
