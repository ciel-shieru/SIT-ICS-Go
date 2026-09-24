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

func TestIsAllowedOrigin(t *testing.T) {
	tests := []struct {
		url    string
		expect bool
		desc   string
	}{
		// Exact matches - all allowed origins
		{"https://in4sit.singaporetech.edu.sg", true, "exact match in4sit"},
		{"https://fs.singaporetech.edu.sg", true, "exact match fs"},
		{"https://xsite.singaporetech.edu.sg", true, "exact match xsite"},

		// Subpath matches - should be allowed (prefix + '/')
		{"https://in4sit.singaporetech.edu.sg/psc/CSSISSTD", true, "in4sit with subpath"},
		{"https://fs.singaporetech.edu.sg/auth", true, "fs with subpath"},
		{"https://xsite.singaporetech.edu.sg/courses", true, "xsite with subpath"},
		{"https://in4sit.singaporetech.edu.sg/", true, "in4sit trailing slash"},
		{"https://fs.singaporetech.edu.sg/", true, "fs trailing slash"},

		// Port suffix - should be allowed (prefix + ':')
		{"https://in4sit.singaporetech.edu.sg:8443", true, "in4sit with port"},
		{"https://fs.singaporetech.edu.sg:9443", true, "fs with port"},

		// Attack vectors - substring domain spoofing (HIGH-2 regression)
		{"https://singaporetech.edu.sg.evil.com", false, "substring attack: evil suffix"},
		{"https://evil-singaporetech.edu.sg", false, "attack: evil prefix"},
		{"https://in4sit.singaporetech.edu.sg.evil.com", false, "attack: in4sit subdomain spoofed"},
		{"https://in4sit.singaporetech.edu.sg.attacker.com", false, "attack: in4sit full domain spoofed"},
		{"https://fs.singaporetech.edu.sg.fake.org", false, "attack: fs full domain spoofed"},
		{"https://xsite.singaporetech.edu.sg.malware.net", false, "attack: xsite full domain spoofed"},
		{"https://notsingaporetech.edu.sg", false, "attack: not prefix"},
		{"https://mysingaporetech.edu.sg", false, "attack: my prefix"},
		{"http://in4sit.singaporetech.edu.sg", false, "scheme mismatch: http instead of https"},
		{"https://in4sit.singaporetech.edu.sg", true, "exact match (same as allowed origin)"},
		{"https://in4sit.singaporetech.edu", false, "attack: missing .sg"},
		{"https://in4sit.singaporetech.edu.sg.evil.com/path", false, "attack: subdomain spoof with path"},
		{"https://in4sit.singaporetech.edu.sg:8443/path?query=value", true, "in4sit with port and query string"},

		// Completely unrelated domains
		{"https://example.com", false, "unrelated domain"},
		{"https://evil.com", false, "evil domain"},
		{"https://singaporetech.edu.sg", false, "missing subdomain (no in4sit/fs/xsite prefix)"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isAllowedOrigin(tt.url)
			if result != tt.expect {
				t.Errorf("isAllowedOrigin(%q) = %v, want %v (%s)", tt.url, result, tt.expect, tt.desc)
			}
		})
	}
}

func TestIsAllowedOriginNoSubstringVulnerability(t *testing.T) {
	// Regression test for HIGH-2: ensure no substring matching allows domain spoofing.
	// Previously, strings.Contains(domain, "singaporetech.edu.sg") would match
	// singaporetech.edu.sg.evil.com — this test prevents regression.

	attackURLs := []string{
		"singaporetech.edu.sg.evil.com",
		"fake-singaporetech.edu.sg",
		"evil.in4sit.singaporetech.edu.sg.evil.com",
		"in4sit.singaporetech.edu.sg.badsite.org",
		"xsite.singaporetech.edu.sg.fake-domain.net",
	}

	for _, url := range attackURLs {
		t.Run(url, func(t *testing.T) {
			// Test with https:// prefix since isAllowedOrigin expects full URLs
			fullURL := "https://" + url
			if isAllowedOrigin(fullURL) {
				t.Errorf("isAllowedOrigin(%q) should reject substring domain attack", fullURL)
			}
		})
	}
}


