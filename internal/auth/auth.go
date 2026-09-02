package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

type Provider interface {
	Authenticate(ctx context.Context, req AuthRequest) (AuthResult, error)
}

type PeoplesoftClient interface {
	SetCookies(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error)
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
	ps      PeoplesoftClient
}

func NewADFSProvider(browser browser.AuthBrowser, ps PeoplesoftClient) *ADFSProvider {
	return &ADFSProvider{
		browser: browser,
		ps:      ps,
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

	psCtx, psCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer psCancel()

	cookies, token, err := a.ps.SetCookies(psCtx, authResult.Cookies)
	if err != nil {
		return AuthResult{}, fmt.Errorf("peoplesoft set cookies: %w", err)
	}

	return AuthResult{
		Cookies: cookies,
		Token:   token,
	}, nil
}
