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
					Cookies:      []Cookie{{Name: "test", Value: "value"}},
					Token:        "token123",
					SAMLResponse: "test-saml-response",
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
		if result.SAMLResponse != "test-saml-response" {
			t.Errorf("expected SAMLResponse 'test-saml-response', got '%s'", result.SAMLResponse)
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

func TestAuthResultSAMLResponse(t *testing.T) {
	result := AuthResult{
		Cookies:      []Cookie{{Name: "PS_TOKEN", Value: "abc"}},
		RedirectURL:  "https://in4sit.singaporetech.edu.sg/psc/",
		SAMLResponse: "abcdefghijkLMNOPQRSTUVwX",
	}

	if result.SAMLResponse != "abcdefghijkLMNOPQRSTUVwX" {
		t.Errorf("expected SAMLResponse 'abcdefghijkLMNOPQRSTUVwX', got '%s'", result.SAMLResponse)
	}
	if len(result.Cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(result.Cookies))
	}
}

func TestDefaultBrowserConfig(t *testing.T) {
	cfg := DefaultBrowserConfig(BrowserAuto)

	if cfg.Mode != BrowserAuto {
		t.Errorf("expected mode %s, got %s", BrowserAuto, cfg.Mode)
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
