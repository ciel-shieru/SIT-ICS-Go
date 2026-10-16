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

		// A browser operation can return nil after its context deadline has
		// already expired (for example, a navigation that completes late under
		// upstream load). Never accept that as success: callers may use this
		// error to retry the complete fetch under a still-live parent context.
		if ctxErr := ctx.Err(); ctxErr != nil {
			if lastErr != nil {
				return fmt.Errorf("retry aborted: operation did not finish before context expired: %w", errors.Join(ctxErr, lastErr))
			}
			return fmt.Errorf("retry aborted: operation returned after context expiration: %w", ctxErr)
		}

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
