package probe

import (
	"testing"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func TestLogProbe_InvalidRegex(t *testing.T) {
	logger := logging.Default()
	config := Config{}

	// Invalid regex should result in nil pattern and failure on Execute
	probe := NewLogProbe(nil, "container123", "[invalid", 30, config, logger)
	if probe.pattern != nil {
		t.Error("expected nil pattern for invalid regex")
	}

	result := probe.Execute(nil)
	if result.Success {
		t.Error("expected failure for invalid regex")
	}
	if result.Type != state.ProbeTypeLog {
		t.Errorf("expected type=log, got %s", result.Type)
	}
	if result.Message != "invalid regex pattern" {
		t.Errorf("expected 'invalid regex pattern', got %q", result.Message)
	}
}

func TestLogProbe_DefaultWindow(t *testing.T) {
	logger := logging.Default()
	probe := NewLogProbe(nil, "container", "ready", 0, Config{}, logger)
	if probe.windowSec != 30 {
		t.Errorf("expected default window=30, got %d", probe.windowSec)
	}
}

func TestLogProbe_Type(t *testing.T) {
	probe := &LogProbe{}
	if probe.Type() != state.ProbeTypeLog {
		t.Errorf("expected type=log, got %s", probe.Type())
	}
}
