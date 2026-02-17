package discovery

import (
	"testing"

	"github.com/itsmrshow/bulwark/internal/state"
)

func TestDeduplicateTargets(t *testing.T) {
	d := &Discoverer{}

	targets := []state.Target{
		{ID: "a", Name: "alpha"},
		{ID: "b", Name: "beta"},
		{ID: "a", Name: "alpha-dup"},
	}

	deduped := d.deduplicateTargets(targets)
	if len(deduped) != 2 {
		t.Errorf("expected 2 unique targets, got %d", len(deduped))
	}

	// First occurrence should be kept
	if deduped[0].Name != "alpha" {
		t.Errorf("expected first target name=alpha, got %s", deduped[0].Name)
	}
}

func TestDeduplicateTargets_Empty(t *testing.T) {
	d := &Discoverer{}
	deduped := d.deduplicateTargets(nil)
	if len(deduped) != 0 {
		t.Errorf("expected 0 targets, got %d", len(deduped))
	}
}

func TestCountServicesByPolicy(t *testing.T) {
	targets := []state.Target{
		{
			Services: []state.Service{
				{Labels: state.Labels{Enabled: true, Policy: state.PolicySafe}},
				{Labels: state.Labels{Enabled: true, Policy: state.PolicyNotify}},
				{Labels: state.Labels{Enabled: false, Policy: state.PolicySafe}},
			},
		},
	}

	counts := CountServicesByPolicy(targets)
	if counts[state.PolicySafe] != 1 {
		t.Errorf("expected 1 safe service, got %d", counts[state.PolicySafe])
	}
	if counts[state.PolicyNotify] != 1 {
		t.Errorf("expected 1 notify service, got %d", counts[state.PolicyNotify])
	}
}

func TestCountServicesByTier(t *testing.T) {
	targets := []state.Target{
		{
			Services: []state.Service{
				{Labels: state.Labels{Enabled: true, Tier: state.TierStateless}},
				{Labels: state.Labels{Enabled: true, Tier: state.TierStateful}},
			},
		},
	}

	counts := CountServicesByTier(targets)
	if counts[state.TierStateless] != 1 {
		t.Errorf("expected 1 stateless, got %d", counts[state.TierStateless])
	}
	if counts[state.TierStateful] != 1 {
		t.Errorf("expected 1 stateful, got %d", counts[state.TierStateful])
	}
}

func TestGetEnabledServices(t *testing.T) {
	targets := []state.Target{
		{
			Services: []state.Service{
				{Name: "web", Labels: state.Labels{Enabled: true}},
				{Name: "db", Labels: state.Labels{Enabled: false}},
			},
		},
	}

	enabled := GetEnabledServices(targets)
	if len(enabled) != 1 {
		t.Errorf("expected 1 enabled service, got %d", len(enabled))
	}
	if enabled[0].Service.Name != "web" {
		t.Errorf("expected service name=web, got %s", enabled[0].Service.Name)
	}
}

func TestGetContainerName(t *testing.T) {
	tests := []struct {
		names    []string
		expected string
	}{
		{[]string{"/mycontainer"}, "mycontainer"},
		{[]string{"noprefix"}, "noprefix"},
		{nil, "unknown"},
		{[]string{}, "unknown"},
	}

	for _, tt := range tests {
		got := getContainerName(tt.names)
		if got != tt.expected {
			t.Errorf("getContainerName(%v) = %q, want %q", tt.names, got, tt.expected)
		}
	}
}
