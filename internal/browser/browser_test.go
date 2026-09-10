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
					RedirectURL: "https://example.com",
					Token:       "token123",
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
		if result.Token != "token123" {
			t.Errorf("expected token 'token123', got '%s'", result.Token)
		}
		if result.RedirectURL != "https://example.com" {
			t.Errorf("expected redirect URL 'https://example.com', got '%s'", result.RedirectURL)
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


