package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/peoplesoft"
)

func TestADFSProviderSuccess(t *testing.T) {
	mockBrowser := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{
				Cookies: []browser.Cookie{
					{Name: "PS_TOKEN", Value: "token123"},
					{Name: "PSJSESSIONID", Value: "session456"},
				},
				RedirectURL: "https://in4sit.singaporetech.edu.sg/psc/",
			}, nil
		},
	}

	ps := &mockPeoplesoftClient{
		setCookiesFunc: func(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error) {
			return cookies, "token123", nil
		},
	}

	provider := NewADFSProvider(mockBrowser, ps)

	result, err := provider.Authenticate(context.Background(), AuthRequest{
		Username:   "testuser",
		Password:   "testpass",
		TOTPSecret: "TOTPSECRET",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(result.Cookies))
	}
	if result.Token != "token123" {
		t.Errorf("expected token 'token123', got '%s'", result.Token)
	}
}

func TestADFSProviderBrowserError(t *testing.T) {
	mockBrowser := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{}, fmt.Errorf("%w: %v", browser.ErrAuthenticationTimeout, context.DeadlineExceeded)
		},
	}

	ps := &mockPeoplesoftClient{}

	provider := NewADFSProvider(mockBrowser, ps)

	_, err := provider.Authenticate(context.Background(), AuthRequest{
		Username: "testuser",
		Password: "testpass",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, browser.ErrAuthenticationTimeout) {
		t.Errorf("expected ErrAuthenticationTimeout, got %v", err)
	}
}

func TestADFSProviderPeoplesoftError(t *testing.T) {
	mockBrowser := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{
				Cookies: []browser.Cookie{{Name: "PS_TOKEN", Value: "token123"}},
			}, nil
		},
	}

	ps := &mockPeoplesoftClient{
		setCookiesFunc: func(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error) {
			return nil, "", fmt.Errorf("peoplesoft error")
		},
	}

	provider := NewADFSProvider(mockBrowser, ps)

	_, err := provider.Authenticate(context.Background(), AuthRequest{
		Username: "testuser",
		Password: "testpass",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type mockPeoplesoftClient struct {
	setCookiesFunc func(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error)
	fetchFunc      func(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error)
}

func (m *mockPeoplesoftClient) SetCookies(ctx context.Context, cookies []browser.Cookie) ([]browser.Cookie, string, error) {
	if m.setCookiesFunc != nil {
		return m.setCookiesFunc(ctx, cookies)
	}
	return cookies, "", nil
}

func (m *mockPeoplesoftClient) FetchTimetable(ctx context.Context, weekDate string) ([]peoplesoft.Entry, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, weekDate)
	}
	return []peoplesoft.Entry{}, nil
}
