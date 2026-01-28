package discovery

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yourusername/bulwark/internal/state"
)

const (
	// Label keys
	LabelEnabled         = "bulwark.enabled"
	LabelPolicy          = "bulwark.policy"
	LabelTier            = "bulwark.tier"
	LabelDefinition      = "bulwark.definition"
	LabelProbeType       = "bulwark.probe.type"
	LabelProbeURL        = "bulwark.probe.url"
	LabelProbeStatus     = "bulwark.probe.expect_status"
	LabelProbeTCPHost    = "bulwark.probe.tcp_host"
	LabelProbeTCPPort    = "bulwark.probe.tcp_port"
	LabelProbeLogPattern = "bulwark.probe.log_pattern"
	LabelProbeWindowSec  = "bulwark.probe.window_sec"
	LabelProbeStability  = "bulwark.probe.stability_sec"
)

// Known database images that should default to stateful tier
var knownDatabases = []string{
	"postgres", "postgresql",
	"mysql", "mariadb",
	"mongodb", "mongo",
	"redis",
	"elasticsearch", "opensearch",
	"cassandra",
	"couchdb",
	"neo4j",
	"influxdb",
	"timescaledb",
	"cockroachdb",
	"percona",
	"mssql", "sqlserver",
}

// ParseLabels parses Bulwark labels from a label map
func ParseLabels(labels map[string]string, imageName string) state.Labels {
	result := state.DefaultLabels()

	// Check if Bulwark is enabled
	if enabled, ok := labels[LabelEnabled]; ok {
		result.Enabled = strings.ToLower(enabled) == "true"
	}

	// Parse policy
	if policy, ok := labels[LabelPolicy]; ok {
		switch strings.ToLower(policy) {
		case "notify":
			result.Policy = state.PolicyNotify
		case "safe":
			result.Policy = state.PolicySafe
		case "aggressive":
			result.Policy = state.PolicyAggressive
		default:
			result.Policy = state.PolicySafe
		}
	}

	// Parse tier - check if explicitly set first
	tierExplicitlySet := false
	if tier, ok := labels[LabelTier]; ok {
		tierExplicitlySet = true
		switch strings.ToLower(tier) {
		case "stateless":
			result.Tier = state.TierStateless
		case "stateful":
			result.Tier = state.TierStateful
		default:
			result.Tier = state.TierStateless
		}
	}

	// If tier not explicitly set, check if it's a known database image
	if !tierExplicitlySet && IsKnownDatabase(imageName) {
		result.Tier = state.TierStateful
	}

	// Parse definition (for loose containers)
	if definition, ok := labels[LabelDefinition]; ok {
		result.Definition = definition
	}

	// Parse probe configuration
	result.Probe = parseProbeConfig(labels)

	return result
}

// parseProbeConfig parses probe configuration from labels
func parseProbeConfig(labels map[string]string) state.ProbeConfig {
	config := state.ProbeConfig{
		Type:       state.ProbeTypeNone,
		HTTPStatus: 200,
	}

	// Parse probe type
	if probeType, ok := labels[LabelProbeType]; ok {
		switch strings.ToLower(probeType) {
		case "docker":
			config.Type = state.ProbeTypeDocker
		case "http":
			config.Type = state.ProbeTypeHTTP
		case "tcp":
			config.Type = state.ProbeTypeTCP
		case "log":
			config.Type = state.ProbeTypeLog
		case "stability":
			config.Type = state.ProbeTypeStability
		case "none":
			config.Type = state.ProbeTypeNone
		}
	}

	// Parse HTTP probe config
	if url, ok := labels[LabelProbeURL]; ok {
		config.HTTPUrl = url
	}
	if status, ok := labels[LabelProbeStatus]; ok {
		if statusInt, err := strconv.Atoi(status); err == nil {
			config.HTTPStatus = statusInt
		}
	}

	// Parse TCP probe config
	if host, ok := labels[LabelProbeTCPHost]; ok {
		config.TCPHost = host
	}
	if port, ok := labels[LabelProbeTCPPort]; ok {
		if portInt, err := strconv.Atoi(port); err == nil {
			config.TCPPort = portInt
		}
	}

	// Parse log probe config
	if pattern, ok := labels[LabelProbeLogPattern]; ok {
		config.LogPattern = pattern
	}
	if window, ok := labels[LabelProbeWindowSec]; ok {
		if windowInt, err := strconv.Atoi(window); err == nil {
			config.WindowSec = windowInt
		}
	}

	// Parse stability window
	if stability, ok := labels[LabelProbeStability]; ok {
		if stabilityInt, err := strconv.Atoi(stability); err == nil {
			config.StabilitySec = stabilityInt
		}
	}

	return config
}

// IsKnownDatabase checks if an image name matches a known database
func IsKnownDatabase(imageName string) bool {
	// Extract the image name without registry and tag
	parts := strings.Split(imageName, "/")
	name := parts[len(parts)-1]

	// Remove tag
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}

	// Remove digest
	if idx := strings.Index(name, "@"); idx >= 0 {
		name = name[:idx]
	}

	name = strings.ToLower(name)

	for _, db := range knownDatabases {
		if strings.Contains(name, db) {
			return true
		}
	}

	return false
}

// ParseDefinition parses the bulwark.definition label
// Format: "compose:/path/to/docker-compose.yml:service-name"
func ParseDefinition(definition string) (path string, service string, err error) {
	if definition == "" {
		return "", "", fmt.Errorf("definition is empty")
	}

	if !strings.HasPrefix(definition, "compose:") {
		return "", "", fmt.Errorf("invalid definition format: must start with 'compose:'")
	}

	// Remove "compose:" prefix
	definition = strings.TrimPrefix(definition, "compose:")

	// Split by the last colon to separate path and service
	parts := strings.Split(definition, ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid definition format: missing service name")
	}

	// The service name is the last part
	service = parts[len(parts)-1]

	// The path is everything before the last colon
	path = strings.Join(parts[:len(parts)-1], ":")

	if path == "" || service == "" {
		return "", "", fmt.Errorf("invalid definition format: empty path or service")
	}

	return path, service, nil
}

// ValidateLabels checks if labels are valid and sufficient for management
func ValidateLabels(labels state.Labels) []string {
	var warnings []string

	if !labels.Enabled {
		warnings = append(warnings, "bulwark.enabled is not set to true")
		return warnings // Early return if not enabled
	}

	// Check probe configuration based on type
	switch labels.Probe.Type {
	case state.ProbeTypeHTTP:
		if labels.Probe.HTTPUrl == "" {
			warnings = append(warnings, "HTTP probe configured but no URL provided")
		}
	case state.ProbeTypeTCP:
		if labels.Probe.TCPHost == "" || labels.Probe.TCPPort == 0 {
			warnings = append(warnings, "TCP probe configured but host or port missing")
		}
	case state.ProbeTypeLog:
		if labels.Probe.LogPattern == "" {
			warnings = append(warnings, "Log probe configured but no pattern provided")
		}
	}

	// Check policy and tier combination
	if labels.Tier == state.TierStateful && labels.Policy == state.PolicyAggressive {
		warnings = append(warnings, "aggressive policy on stateful service is risky")
	}

	return warnings
}
