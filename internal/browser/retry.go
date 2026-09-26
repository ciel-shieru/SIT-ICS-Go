package browser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type RetryableFunc func() error

func isRetriable(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, ErrAuthentication) ||
		errors.Is(err, ErrAuthenticationTimeout) ||
		errors.Is(err, ErrCredentialExtraction) {
		return false
	}
	return true
}

func Do(ctx context.Context, fn RetryableFunc, maxRetries int, interval time.Duration) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry aborted: %w", ctx.Err())
			case <-time.After(interval):
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("panic: %v", r)
				}
			}()
			lastErr = fn()
		}()

		if lastErr == nil {
			return nil
		}

		if !isRetriable(lastErr) {
			return lastErr
		}

		log.Printf("browser: attempt %d/%d failed: %v", attempt+1, maxRetries+1, lastErr)
	}
	return fmt.Errorf("retry failed after %d attempts: %w", maxRetries+1, lastErr)
}
