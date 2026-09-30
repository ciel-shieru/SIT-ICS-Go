//go:build !container

package credretry

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ciel-shieru/sit-ics-go/internal/browser"
	"github.com/ciel-shieru/sit-ics-go/internal/config"
	"github.com/ciel-shieru/sit-ics-go/internal/credprompt"
)

// PromptIfAuthFailed checks if the error is credential-related and, if so,
// prompts the user to re-enter credentials interactively. Returns nil on
// successful retry or if the user declines; returns the original error
// if prompting fails or the error is not credential-related.
func PromptIfAuthFailed(cfg *config.Config, err error) error {
	if !isCredentialError(err) {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	if _, err := os.Stdout.Write([]byte("Authentication failed. Re-enter credentials? [Y/n] ")); err != nil {
		return fmt.Errorf("write prompt: %w", err)
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	choice := strings.TrimSpace(strings.ToLower(line))
	if choice == "n" || choice == "no" || choice == "" {
		return err // user declined or pressed Enter — continue with failure
	}

	// User answered Y/yes — re-prompt for credentials (overwrites old ones)
	if promptErr := credprompt.PromptIfNeeded(cfg, true); promptErr != nil {
		return fmt.Errorf("credential re-prompt failed: %w", promptErr)
	}

	return nil
}

// isCredentialError returns true if the error indicates invalid credentials
// or TOTP generation failure (as opposed to network/browser errors).
func isCredentialError(err error) bool {
	if err == nil {
		return false
	}

	// Check sentinel errors
	if errors.Is(err, browser.ErrAuthentication) ||
		errors.Is(err, browser.ErrCredentialExtraction) {
		return true
	}

	// Check for ADFS-specific error message (auth.go:180-182)
	if strings.Contains(err.Error(), "ADFS auth error:") {
		return true
	}

	return false
}
