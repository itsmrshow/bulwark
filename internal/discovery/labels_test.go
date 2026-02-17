package discovery

import (
	"testing"

	"github.com/itsmrshow/bulwark/internal/state"
)

func TestParseLabels_Defaults(t *testing.T) {
	labels := ParseLabels(map[string]string{}, "nginx:latest")
	if labels.Enabled {
		t.Error("expected enabled=false by default")
	}
	if labels.Policy != state.PolicySafe {
		t.Errorf("expected policy=safe, got %s", labels.Policy)
	}
	if labels.Tier != state.TierStateless {
		t.Errorf("expected tier=stateless, got %s", labels.Tier)
	}
	if labels.Probe.Type != state.ProbeTypeNone {
		t.Errorf("expected probe type=none, got %s", labels.Probe.Type)
	}
}

func TestParseLabels_Enabled(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled": "true",
	}, "nginx:latest")
	if !labels.Enabled {
		t.Error("expected enabled=true")
	}
}

func TestParseLabels_AllPolicies(t *testing.T) {
	tests := []struct {
		value    string
		expected state.Policy
	}{
		{"notify", state.PolicyNotify},
		{"safe", state.PolicySafe},
		{"aggressive", state.PolicyAggressive},
		{"unknown", state.PolicySafe},
	}

	for _, tt := range tests {
		labels := ParseLabels(map[string]string{
			"bulwark.enabled": "true",
			"bulwark.policy":  tt.value,
		}, "nginx")
		if labels.Policy != tt.expected {
			t.Errorf("policy=%q: expected %s, got %s", tt.value, tt.expected, labels.Policy)
		}
	}
}

func TestParseLabels_Tier(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled": "true",
		"bulwark.tier":    "stateful",
	}, "nginx")
	if labels.Tier != state.TierStateful {
		t.Errorf("expected tier=stateful, got %s", labels.Tier)
	}
}

func TestParseLabels_ProbeHTTP(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled":             "true",
		"bulwark.probe.type":          "http",
		"bulwark.probe.url":           "http://localhost:8080/health",
		"bulwark.probe.expect_status": "200",
	}, "nginx")
	if labels.Probe.Type != state.ProbeTypeHTTP {
		t.Errorf("expected probe type=http, got %s", labels.Probe.Type)
	}
	if labels.Probe.HTTPUrl != "http://localhost:8080/health" {
		t.Errorf("expected URL, got %s", labels.Probe.HTTPUrl)
	}
	if labels.Probe.HTTPStatus != 200 {
		t.Errorf("expected status=200, got %d", labels.Probe.HTTPStatus)
	}
}

func TestParseLabels_ProbeTCP(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled":        "true",
		"bulwark.probe.type":     "tcp",
		"bulwark.probe.tcp_host": "localhost",
		"bulwark.probe.tcp_port": "5432",
	}, "postgres")
	if labels.Probe.Type != state.ProbeTypeTCP {
		t.Errorf("expected probe type=tcp, got %s", labels.Probe.Type)
	}
	if labels.Probe.TCPHost != "localhost" {
		t.Errorf("expected host=localhost, got %s", labels.Probe.TCPHost)
	}
	if labels.Probe.TCPPort != 5432 {
		t.Errorf("expected port=5432, got %d", labels.Probe.TCPPort)
	}
}

func TestParseLabels_ProbeLog(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled":           "true",
		"bulwark.probe.type":        "log",
		"bulwark.probe.log_pattern": "ready to accept connections",
		"bulwark.probe.window_sec":  "60",
	}, "postgres")
	if labels.Probe.Type != state.ProbeTypeLog {
		t.Errorf("expected probe type=log, got %s", labels.Probe.Type)
	}
	if labels.Probe.LogPattern != "ready to accept connections" {
		t.Errorf("expected pattern, got %s", labels.Probe.LogPattern)
	}
	if labels.Probe.WindowSec != 60 {
		t.Errorf("expected window_sec=60, got %d", labels.Probe.WindowSec)
	}
}

func TestParseLabels_ProbeStability(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled":             "true",
		"bulwark.probe.type":          "stability",
		"bulwark.probe.stability_sec": "30",
	}, "app")
	if labels.Probe.Type != state.ProbeTypeStability {
		t.Errorf("expected probe type=stability, got %s", labels.Probe.Type)
	}
	if labels.Probe.StabilitySec != 30 {
		t.Errorf("expected stability_sec=30, got %d", labels.Probe.StabilitySec)
	}
}

func TestParseLabels_DatabaseAutoStateful(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled": "true",
	}, "postgres:15")
	if labels.Tier != state.TierStateful {
		t.Errorf("expected tier=stateful for database image, got %s", labels.Tier)
	}
}

func TestParseLabels_ExplicitTierOverridesDatabase(t *testing.T) {
	labels := ParseLabels(map[string]string{
		"bulwark.enabled": "true",
		"bulwark.tier":    "stateless",
	}, "postgres:15")
	if labels.Tier != state.TierStateless {
		t.Errorf("explicit tier should override database detection, got %s", labels.Tier)
	}
}

func TestIsKnownDatabase(t *testing.T) {
	tests := []struct {
		image    string
		expected bool
	}{
		{"postgres:15", true},
		{"mysql:8", true},
		{"redis:7", true},
		{"mongo:6", true},
		{"mariadb:10", true},
		{"nginx:latest", false},
		{"ghcr.io/user/postgres-proxy:v1", true},
		{"app:latest", false},
		{"elasticsearch:8", true},
	}

	for _, tt := range tests {
		if got := IsKnownDatabase(tt.image); got != tt.expected {
			t.Errorf("IsKnownDatabase(%q) = %v, want %v", tt.image, got, tt.expected)
		}
	}
}

func TestValidateLabels(t *testing.T) {
	// Not enabled
	warnings := ValidateLabels(state.Labels{Enabled: false})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for disabled, got %d", len(warnings))
	}

	// HTTP probe without URL
	warnings = ValidateLabels(state.Labels{
		Enabled: true,
		Probe:   state.ProbeConfig{Type: state.ProbeTypeHTTP},
	})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for HTTP probe without URL, got %d", len(warnings))
	}

	// TCP probe without host/port
	warnings = ValidateLabels(state.Labels{
		Enabled: true,
		Probe:   state.ProbeConfig{Type: state.ProbeTypeTCP},
	})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for TCP probe without host/port, got %d", len(warnings))
	}

	// Log probe without pattern
	warnings = ValidateLabels(state.Labels{
		Enabled: true,
		Probe:   state.ProbeConfig{Type: state.ProbeTypeLog},
	})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for log probe without pattern, got %d", len(warnings))
	}

	// Aggressive + stateful
	warnings = ValidateLabels(state.Labels{
		Enabled: true,
		Policy:  state.PolicyAggressive,
		Tier:    state.TierStateful,
		Probe:   state.ProbeConfig{Type: state.ProbeTypeNone},
	})
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for aggressive+stateful, got %d", len(warnings))
	}

	// Valid config
	warnings = ValidateLabels(state.Labels{
		Enabled: true,
		Policy:  state.PolicySafe,
		Tier:    state.TierStateless,
		Probe: state.ProbeConfig{
			Type:    state.ProbeTypeHTTP,
			HTTPUrl: "http://localhost/health",
		},
	})
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestParseDefinition(t *testing.T) {
	tests := []struct {
		name       string
		definition string
		wantPath   string
		wantSvc    string
		wantErr    bool
	}{
		{"valid", "compose:/docker_data/app/docker-compose.yml#service=web", "/docker_data/app/docker-compose.yml", "web", false},
		{"empty", "", "", "", true},
		{"no_prefix", "/path/compose.yml#service=web", "", "", true},
		{"no_fragment", "compose:/path/compose.yml", "", "", true},
		{"no_service_value", "compose:/path/compose.yml#service=", "", "", true},
		{"wrong_fragment", "compose:/path/compose.yml#svc=web", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, svc, err := ParseDefinition(tt.definition)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDefinition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
			if svc != tt.wantSvc {
				t.Errorf("service = %q, want %q", svc, tt.wantSvc)
			}
		})
	}
}
