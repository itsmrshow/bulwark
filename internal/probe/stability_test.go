package probe

import (
	"context"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func TestStabilityProbe_ShortWindow(t *testing.T) {
	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	probe := NewStabilityProbe(1, config, logger) // 1 second window
	result := probe.Execute(context.Background())

	if !result.Success {
		t.Errorf("expected success for 1s stability window, got: %s", result.Message)
	}
	if result.Type != state.ProbeTypeStability {
		t.Errorf("expected type=stability, got %s", result.Type)
	}
}

func TestStabilityProbe_ContextCancel(t *testing.T) {
	logger := logging.Default()
	config := Config{
		Timeout:      5 * time.Second,
		Retries:      1,
		RetryBackoff: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	probe := NewStabilityProbe(60, config, logger) // 60 second window, but context cancels after 100ms
	result := probe.Execute(ctx)

	if result.Success {
		t.Error("expected failure when context cancelled before window")
	}
}

func TestStabilityProbe_DefaultWindow(t *testing.T) {
	probe := NewStabilityProbe(0, Config{}, logging.Default())
	if probe.windowSec != 10 {
		t.Errorf("expected default window=10, got %d", probe.windowSec)
	}
}
