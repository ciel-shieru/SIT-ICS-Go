//go:build !container

package credretry

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
)

func TestIsCredentialError_ErrAuthentication(t *testing.T) {
	if !isCredentialError(browser.ErrAuthentication) {
		t.Error("expected true for ErrAuthentication")
	}
}

func TestIsCredentialError_ErrAuthenticationWrapped(t *testing.T) {
	wrapped := fmt.Errorf("adfs authenticate: %w", browser.ErrAuthentication)
	if !isCredentialError(wrapped) {
		t.Error("expected true for wrapped ErrAuthentication")
	}
}

func TestIsCredentialError_ADFSLoginError(t *testing.T) {
	adfsErr := errors.New("ADFS auth error: Username or password is incorrect")
	if !isCredentialError(adfsErr) {
		t.Error("expected true for ADFS auth error")
	}
}

func TestIsCredentialError_TOTPGenerationFailure(t *testing.T) {
	totpErr := fmt.Errorf("%w: failed to generate TOTP: invalid secret", browser.ErrAuthentication)
	if !isCredentialError(totpErr) {
		t.Error("expected true for TOTP generation failure")
	}
}

func TestIsCredentialError_NetworkError(t *testing.T) {
	if isCredentialError(browser.ErrBrowserConnect) {
		t.Error("expected false for ErrBrowserConnect")
	}
}

func TestIsCredentialError_NavigationError(t *testing.T) {
	if isCredentialError(browser.ErrNavigation) {
		t.Error("expected false for ErrNavigation")
	}
}

func TestIsCredentialError_CredentialExtraction(t *testing.T) {
	if !isCredentialError(browser.ErrCredentialExtraction) {
		t.Error("expected true for ErrCredentialExtraction")
	}

	wrapped := fmt.Errorf("some wrapper: %w", browser.ErrCredentialExtraction)
	if !isCredentialError(wrapped) {
		t.Error("expected true for wrapped ErrCredentialExtraction")
	}
}

func TestIsCredentialError_GenericError(t *testing.T) {
	if isCredentialError(errors.New("some random error")) {
		t.Error("expected false for generic error")
	}
}

func TestIsCredentialError_Nil(t *testing.T) {
	if isCredentialError(nil) {
		t.Error("expected false for nil")
	}
}
