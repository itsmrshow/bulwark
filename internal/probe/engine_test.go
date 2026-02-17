package probe

import (
	"testing"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func testEngine() *Engine {
	logger := logging.New(logging.Config{Level: "error"})
	return NewEngine(nil, logger)
}

func TestAllProbesPassed_Empty(t *testing.T) {
	if !AllProbesPassed(nil) {
		t.Error("expected true for nil results")
	}
	if !AllProbesPassed([]state.ProbeResult{}) {
		t.Error("expected true for empty results")
	}
}

func TestAllProbesPassed_AllPass(t *testing.T) {
	results := []state.ProbeResult{
		{Type: state.ProbeTypeHTTP, Success: true},
		{Type: state.ProbeTypeTCP, Success: true},
	}
	if !AllProbesPassed(results) {
		t.Error("expected true when all pass")
	}
}

func TestAllProbesPassed_OneFails(t *testing.T) {
	results := []state.ProbeResult{
		{Type: state.ProbeTypeHTTP, Success: true},
		{Type: state.ProbeTypeTCP, Success: false},
	}
	if AllProbesPassed(results) {
		t.Error("expected false when one fails")
	}
}

func TestCollectProbes_None(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{Type: state.ProbeTypeNone}, "")
	if len(probes) != 0 {
		t.Errorf("expected 0 probes for none type, got %d", len(probes))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors for none type, got %d", len(errors))
	}
}

func TestCollectProbes_UnknownType(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{Type: "unknown"}, "")
	if len(probes) != 0 {
		t.Errorf("expected 0 probes for unknown type, got %d", len(probes))
	}
	if len(errors) != 1 {
		t.Errorf("expected 1 error for unknown type, got %d", len(errors))
	}
}

func TestCollectProbes_HTTPMissingURL(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{Type: state.ProbeTypeHTTP}, "")
	if len(probes) != 0 {
		t.Errorf("expected 0 probes, got %d", len(probes))
	}
	if len(errors) != 1 || errors[0].Type != state.ProbeTypeHTTP {
		t.Error("expected HTTP error result")
	}
}

func TestCollectProbes_TCPMissingConfig(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{Type: state.ProbeTypeTCP}, "")
	if len(probes) != 0 {
		t.Errorf("expected 0 probes, got %d", len(probes))
	}
	if len(errors) != 1 || errors[0].Type != state.ProbeTypeTCP {
		t.Error("expected TCP error result")
	}
}

func TestCollectProbes_LogMissingPattern(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{Type: state.ProbeTypeLog}, "")
	if len(probes) != 0 {
		t.Errorf("expected 0 probes, got %d", len(probes))
	}
	if len(errors) != 1 || errors[0].Type != state.ProbeTypeLog {
		t.Error("expected log error result")
	}
}

func TestCollectProbes_HTTP(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{
		Type:       state.ProbeTypeHTTP,
		HTTPUrl:    "http://localhost/health",
		HTTPStatus: 200,
	}, "")
	if len(probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(probes))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

func TestCollectProbes_TCP(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{
		Type:    state.ProbeTypeTCP,
		TCPHost: "localhost",
		TCPPort: 5432,
	}, "")
	if len(probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(probes))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

func TestCollectProbes_Stability(t *testing.T) {
	engine := testEngine()
	probes, errors := engine.collectProbes(state.ProbeConfig{
		Type:         state.ProbeTypeStability,
		StabilitySec: 10,
	}, "")
	if len(probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(probes))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}
