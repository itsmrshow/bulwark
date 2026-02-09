package probe

import (
	"context"
	"fmt"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

// StabilityProbe waits for a stability window without restarts
type StabilityProbe struct {
	windowSec int
	config    Config
	logger    *logging.Logger
}

// NewStabilityProbe creates a new stability probe
func NewStabilityProbe(windowSec int, config Config, logger *logging.Logger) *StabilityProbe {
	if windowSec == 0 {
		windowSec = 10 // Default 10 seconds
	}

	return &StabilityProbe{
		windowSec: windowSec,
		config:    config,
		logger:    logger.WithComponent("stability-probe"),
	}
}

// Type returns the probe type
func (p *StabilityProbe) Type() state.ProbeType {
	return state.ProbeTypeStability
}

// Execute runs the stability probe
func (p *StabilityProbe) Execute(ctx context.Context) *state.ProbeResult {
	p.logger.Debug().
		Int("window_sec", p.windowSec).
		Msg("Starting stability probe")

	start := time.Now()
	duration := time.Duration(p.windowSec) * time.Second

	// Create a timer for the stability window
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Successfully waited the full window
		elapsed := time.Since(start)
		message := fmt.Sprintf("stable for %d seconds", p.windowSec)

		p.logger.Info().
			Int("window_sec", p.windowSec).
			Dur("duration", elapsed).
			Msg("Stability probe succeeded")

		return &state.ProbeResult{
			Type:     p.Type(),
			Success:  true,
			Duration: elapsed,
			Message:  message,
		}

	case <-ctx.Done():
		// Context canceled before window elapsed
		elapsed := time.Since(start)
		message := fmt.Sprintf("stability window interrupted after %v (needed %ds)", elapsed, p.windowSec)

		p.logger.Warn().
			Int("window_sec", p.windowSec).
			Dur("duration", elapsed).
			Msg("Stability probe interrupted")

		return &state.ProbeResult{
			Type:     p.Type(),
			Success:  false,
			Duration: elapsed,
			Message:  message,
		}
	}
}
