package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
)

func TestADFSProviderSuccess(t *testing.T) {
	mockBrowser := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{
				Cookies: []browser.Cookie{
					{Name: "PS_TOKEN", Value: "token123", Domain: ".singaporetech.edu.sg", Path: "/"},
					{Name: "AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID", Value: "session456", Domain: ".singaporetech.edu.sg", Path: "/"},
					{Name: "MSISAuth", Value: "adfs_cookie", Domain: "fs.singaporetech.edu.sg", Path: "/adfs"},
				},
				RedirectURL: "https://in4sit.singaporetech.edu.sg/psc/CSSISSTD/EMPLOYEE/SA/c/NUI_FRAMEWORK.PT_LANDINGPAGE.GBL",
				SAMLResponse: "test-saml-response-value",
			}, nil
		},
	}

	provider := NewADFSProvider(mockBrowser)

	result, err := provider.Authenticate(context.Background(), AuthRequest{
		Username:   "testuser",
		Password:   "testpass",
		TOTPSecret: "TOTPSECRET",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Cookies) != 3 {
		t.Errorf("expected 3 cookies, got %d", len(result.Cookies))
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

	provider := NewADFSProvider(mockBrowser)

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

func TestADFSProviderNoTokenInCookies(t *testing.T) {
	mockBrowser := &browser.MockAuthBrowser{
		AuthenticateFunc: func(ctx context.Context, req browser.AuthRequest) (browser.AuthResult, error) {
			return browser.AuthResult{
				Cookies: []browser.Cookie{
					{Name: "PSJSESSIONID", Value: "session456", Domain: ".singaporetech.edu.sg"},
				},
				SAMLResponse: "test-saml-response-value",
			}, nil
		},
	}

	provider := NewADFSProvider(mockBrowser)

	result, err := provider.Authenticate(context.Background(), AuthRequest{
		Username: "testuser",
		Password: "testpass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Token != "" {
		t.Errorf("expected empty token, got '%s'", result.Token)
	}
	if len(result.Cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(result.Cookies))
	}
}

func TestExtractPS_TOKEN(t *testing.T) {
	tests := []struct {
		name     string
		cookies  []browser.Cookie
		expected string
	}{
		{
			name: "PS_TOKEN found",
			cookies: []browser.Cookie{
				{Name: "PS_TOKEN", Value: "abc123"},
				{Name: "PSJSESSIONID", Value: "session789"},
			},
			expected: "abc123",
		},
		{
			name: "PS_TOKEN not found",
			cookies: []browser.Cookie{
				{Name: "PSJSESSIONID", Value: "session789"},
			},
			expected: "",
		},
		{
			name:     "empty cookies",
			cookies:  []browser.Cookie{},
			expected: "",
		},
		{
			name: "PS_TOKEN is first",
			cookies: []browser.Cookie{
				{Name: "PS_TOKEN", Value: "first"},
				{Name: "PS_TOKEN", Value: "second"},
			},
			expected: "first",
		},
		{
			name: "PS_TOKEN with domain qualifier",
			cookies: []browser.Cookie{
				{Name: "PS_TOKEN", Value: "domain-token", Domain: ".singaporetech.edu.sg"},
				{Name: "AWSSISWEBPRD02-8002-PORTAL-PSJSESSIONID", Value: "session789", Domain: ".singaporetech.edu.sg"},
			},
			expected: "domain-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPS_TOKEN(tt.cookies)
			if result != tt.expected {
				t.Errorf("extractPS_TOKEN(%v) = %q, want %q", tt.cookies, result, tt.expected)
			}
		})
	}
}
