package browser

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetrySuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return nil
	}, 3, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetrySuccessAfterRetries(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("temporary error")
		}
		return nil
	}, 3, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryExhausted(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return fmt.Errorf("persistent error")
	}, 2, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
	if !containsString(err.Error(), "retry failed after 3 attempts") {
		t.Errorf("error should contain 'retry failed after 3 attempts', got: %v", err)
	}
}

func TestRetryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := Do(ctx, func() error {
		calls++
		return fmt.Errorf("error")
	}, 3, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !containsString(err.Error(), "retry aborted") {
		t.Errorf("error should contain 'retry aborted', got: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

func TestRetryNonRetriableError(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return ErrAuthentication
	}, 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAuthentication) {
		t.Errorf("expected ErrAuthentication, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for non-retriable), got %d", calls)
	}
}

func TestRetryNonRetriableCredentialExtraction(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return ErrCredentialExtraction
	}, 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrCredentialExtraction) {
		t.Errorf("expected ErrCredentialExtraction, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryNonRetriableAuthenticationTimeout(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return ErrAuthenticationTimeout
	}, 3, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrAuthenticationTimeout) {
		t.Errorf("expected ErrAuthenticationTimeout, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryPanicRecovery(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		if calls < 2 {
			panic("test panic")
		}
		return nil
	}, 3, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetryMaxRetriesZero(t *testing.T) {
	calls := 0
	err := Do(context.Background(), func() error {
		calls++
		return fmt.Errorf("error")
	}, 0, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

func TestRetryPreservesLastError(t *testing.T) {
	lastErr := fmt.Errorf("last error")
	err := Do(context.Background(), func() error {
		return lastErr
	}, 2, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, lastErr) {
		t.Errorf("expected wrapped last error, got: %v", err)
	}
}

func TestIsRetriable(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil", nil, true},
		{"ErrAuthentication", ErrAuthentication, false},
		{"ErrAuthentication wrapped", fmt.Errorf("%w: cause", ErrAuthentication), false},
		{"ErrAuthenticationTimeout", ErrAuthenticationTimeout, false},
		{"ErrCredentialExtraction", ErrCredentialExtraction, false},
		{"generic error", fmt.Errorf("some error"), true},
		{"ErrBrowserConnect", ErrBrowserConnect, true},
		{"ErrBrowserLaunch", ErrBrowserLaunch, true},
		{"ErrNavigation", ErrNavigation, true},
		{"ErrBrowserUnavailable", ErrBrowserUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriable(tt.err)
			if result != tt.expect {
				t.Errorf("isRetriable(%v) = %v, want %v", tt.err, result, tt.expect)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
