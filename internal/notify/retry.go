package notify

import (
	"context"
	"fmt"
	"time"
)

const (
	retryAttempts    = 3
	retryBaseBackoff = 1 * time.Second
)

// sendWithRetry retries a webhook send function with exponential backoff.
func sendWithRetry(ctx context.Context, name string, sendFn func(ctx context.Context) (int, error)) error {
	var lastErr error
	backoff := retryBaseBackoff

	for attempt := 0; attempt < retryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("%s: context canceled during retry: %w", name, ctx.Err())
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		statusCode, err := sendFn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx) except 429 (rate limit)
		if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
			return lastErr
		}
	}

	return fmt.Errorf("%s: failed after %d attempts: %w", name, retryAttempts, lastErr)
}
