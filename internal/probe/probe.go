package probe

import (
	"context"
	"time"

	"github.com/itsmrshow/bulwark/internal/state"
)

// Probe defines the interface for health probes
type Probe interface {
	// Execute runs the probe and returns the result
	Execute(ctx context.Context) *state.ProbeResult

	// Type returns the probe type
	Type() state.ProbeType
}

// Config contains common probe configuration
type Config struct {
	Timeout      time.Duration
	Interval     time.Duration
	Retries      int
	RetryBackoff time.Duration
}

// DefaultConfig returns default probe configuration
func DefaultConfig() Config {
	return Config{
		Timeout:      10 * time.Second,
		Interval:     2 * time.Second,
		Retries:      3,
		RetryBackoff: 1 * time.Second,
	}
}

// executeWithRetries executes a probe function with retries
func executeWithRetries(ctx context.Context, config Config, probeFn func(context.Context) error) (bool, time.Duration, string) {
	start := time.Now()
	var lastErr error

	for attempt := 0; attempt < config.Retries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return false, time.Since(start), "context canceled during retry backoff"
			case <-time.After(config.RetryBackoff):
			}
		}

		// Create timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, config.Timeout)
		err := probeFn(attemptCtx)
		cancel()

		if err == nil {
			return true, time.Since(start), "probe succeeded"
		}

		lastErr = err

		// Don't retry if context is canceled
		if ctx.Err() != nil {
			break
		}
	}

	if lastErr != nil {
		return false, time.Since(start), lastErr.Error()
	}

	return false, time.Since(start), "probe failed after all retries"
}
