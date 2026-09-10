package browser

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSentinelErrors(t *testing.T) {
	tests := []error{
		ErrBrowserUnavailable,
		ErrBrowserLaunch,
		ErrBrowserConnect,
		ErrNavigation,
		ErrAuthentication,
		ErrAuthenticationTimeout,
		ErrCredentialExtraction,
	}

	for _, err := range tests {
		if err == nil {
			t.Errorf("expected non-nil error, got nil")
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	underlying := errors.New("underlying cause")
	wrapped := fmt.Errorf("%w: %w", ErrBrowserConnect, underlying)

	if !errors.Is(wrapped, ErrBrowserConnect) {
		t.Error("errors.Is() should find wrapped error")
	}
	if !errors.Is(wrapped, underlying) {
		t.Error("errors.Is() should find underlying error")
	}
}

func TestMockAuthBrowser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &MockAuthBrowser{
			AuthenticateFunc: func(ctx context.Context, req AuthRequest) (AuthResult, error) {
				return AuthResult{
					Cookies: []Cookie{{Name: "test", Value: "value"}},
					Token:   "token123",
				}, nil
			},
		}

		result, err := mock.Authenticate(context.Background(), AuthRequest{
			Username: "user",
			Password: "pass",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Cookies) != 1 {
			t.Errorf("expected 1 cookie, got %d", len(result.Cookies))
		}
		if result.Token != "token123" {
			t.Errorf("expected token 'token123', got '%s'", result.Token)
		}
	})

	t.Run("error", func(t *testing.T) {
		mock := &MockAuthBrowser{
			AuthenticateFunc: func(ctx context.Context, req AuthRequest) (AuthResult, error) {
				return AuthResult{}, fmt.Errorf("%w: %v", ErrAuthenticationTimeout, context.DeadlineExceeded)
			},
		}

		_, err := mock.Authenticate(context.Background(), AuthRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrAuthenticationTimeout) {
			t.Errorf("expected ErrAuthenticationTimeout, got %v", err)
		}
	})

	t.Run("empty cookies", func(t *testing.T) {
		mock := &MockAuthBrowser{
			AuthenticateFunc: func(ctx context.Context, req AuthRequest) (AuthResult, error) {
				return AuthResult{Cookies: nil}, nil
			},
		}

		result, err := mock.Authenticate(context.Background(), AuthRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Cookies != nil {
			t.Error("expected nil cookies")
		}
	})
}

func TestDefaultBrowserConfig(t *testing.T) {
	cfg := DefaultBrowserConfig(BrowserModeAuto)

	if cfg.Mode != BrowserModeAuto {
		t.Errorf("expected mode %s, got %s", BrowserModeAuto, cfg.Mode)
	}
	if !cfg.Headless {
		t.Error("expected headless to be true")
	}
	if !cfg.Incognito {
		t.Error("expected incognito to be true")
	}
	if cfg.ConnectTimeout != 10*time.Second {
		t.Errorf("expected connect timeout 10s, got %v", cfg.ConnectTimeout)
	}
	if cfg.NavigationTimeout != 30*time.Second {
		t.Errorf("expected navigation timeout 30s, got %v", cfg.NavigationTimeout)
	}
	if cfg.AuthTimeout != 5*time.Minute {
		t.Errorf("expected auth timeout 5m, got %v", cfg.AuthTimeout)
	}
}

func TestCookieDeduplication(t *testing.T) {
	cookieMap := make(map[string]Cookie)

	cookies := []Cookie{
		{Name: "PS_TOKEN", Value: "old-token", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 1000},
		{Name: "AWSALB", Value: "cookie1", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 2000},
		{Name: "PS_TOKEN", Value: "new-token", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 3000},
		{Name: "PS_TOKEN", Value: "oldest-token", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 500},
		{Name: "SessionID", Value: "sess1", Domain: "in4sit.singaporetech.edu.sg", Path: "/", Expiry: 4000},
	}

	for _, c := range cookies {
		existing, exists := cookieMap[c.Name]
		if !exists || c.Expiry > existing.Expiry {
			cookieMap[c.Name] = c
		}
	}

	if len(cookieMap) != 3 {
		t.Errorf("expected 3 unique cookies, got %d", len(cookieMap))
	}

	psToken := cookieMap["PS_TOKEN"]
	if psToken.Value != "new-token" {
		t.Errorf("expected PS_TOKEN value 'new-token', got '%s'", psToken.Value)
	}
	if psToken.Expiry != 3000 {
		t.Errorf("expected PS_TOKEN expiry 3000, got %d", psToken.Expiry)
	}

	awsAlb := cookieMap["AWSALB"]
	if awsAlb.Value != "cookie1" {
		t.Errorf("expected AWSALB value 'cookie1', got '%s'", awsAlb.Value)
	}

	sessionID := cookieMap["SessionID"]
	if sessionID.Value != "sess1" {
		t.Errorf("expected SessionID value 'sess1', got '%s'", sessionID.Value)
	}
}

func TestCookieExpiryZeroIsSessionCookie(t *testing.T) {
	cookieMap := make(map[string]Cookie)

	cookies := []Cookie{
		{Name: "SessionCookie", Value: "session-val", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 0},
		{Name: "PersistentCookie", Value: "persist-val", Domain: ".singaporetech.edu.sg", Path: "/", Expiry: 9999999},
	}

	for _, c := range cookies {
		existing, exists := cookieMap[c.Name]
		if !exists || c.Expiry > existing.Expiry {
			cookieMap[c.Name] = c
		}
	}

	if len(cookieMap) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookieMap))
	}

	session := cookieMap["SessionCookie"]
	if session.Expiry != 0 {
		t.Errorf("expected session cookie expiry 0, got %d", session.Expiry)
	}
}
