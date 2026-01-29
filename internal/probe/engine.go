package probe

import (
	"context"
	"fmt"

	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// Engine orchestrates health probes for services
type Engine struct {
	dockerClient *docker.Client
	config       Config
	logger       *logging.Logger
}

// NewEngine creates a new probe engine
func NewEngine(dockerClient *docker.Client, logger *logging.Logger) *Engine {
	return &Engine{
		dockerClient: dockerClient,
		config:       DefaultConfig(),
		logger:       logger.WithComponent("probe-engine"),
	}
}

// ExecuteProbes runs all configured probes for a service
func (e *Engine) ExecuteProbes(ctx context.Context, target *state.Target, service *state.Service, containerID string) []state.ProbeResult {
	probeConfig := service.Labels.Probe

	e.logger.Info().
		Str("service", service.Name).
		Str("probe_type", string(probeConfig.Type)).
		Msg("Executing health probes")

	var results []state.ProbeResult

	// Execute the configured probe type
	switch probeConfig.Type {
	case state.ProbeTypeDocker:
		probe := NewDockerProbe(e.dockerClient, containerID, e.config, e.logger)
		result := probe.Execute(ctx)
		results = append(results, *result)

	case state.ProbeTypeHTTP:
		if probeConfig.HTTPUrl == "" {
			e.logger.Warn().Msg("HTTP probe configured but no URL provided, skipping")
			results = append(results, state.ProbeResult{
				Type:    state.ProbeTypeHTTP,
				Success: false,
				Message: "HTTP probe URL not configured",
			})
		} else {
			probe := NewHTTPProbe(probeConfig.HTTPUrl, probeConfig.HTTPStatus, e.config, e.logger)
			result := probe.Execute(ctx)
			results = append(results, *result)
		}

	case state.ProbeTypeTCP:
		if probeConfig.TCPHost == "" || probeConfig.TCPPort == 0 {
			e.logger.Warn().Msg("TCP probe configured but host/port missing, skipping")
			results = append(results, state.ProbeResult{
				Type:    state.ProbeTypeTCP,
				Success: false,
				Message: "TCP probe host/port not configured",
			})
		} else {
			probe := NewTCPProbe(probeConfig.TCPHost, probeConfig.TCPPort, e.config, e.logger)
			result := probe.Execute(ctx)
			results = append(results, *result)
		}

	case state.ProbeTypeStability:
		stabilityWindow := probeConfig.StabilitySec
		if stabilityWindow == 0 {
			stabilityWindow = 10 // Default 10 seconds
		}
		probe := NewStabilityProbe(stabilityWindow, e.config, e.logger)
		result := probe.Execute(ctx)
		results = append(results, *result)

	case state.ProbeTypeNone:
		// No probe configured, consider it successful
		e.logger.Debug().Msg("No probe configured, skipping health checks")
		return results

	default:
		e.logger.Warn().
			Str("probe_type", string(probeConfig.Type)).
			Msg("Unknown probe type, skipping")
		results = append(results, state.ProbeResult{
			Type:    probeConfig.Type,
			Success: false,
			Message: fmt.Sprintf("unknown probe type: %s", probeConfig.Type),
		})
	}

	// Log summary
	allPassed := true
	for _, result := range results {
		if !result.Success {
			allPassed = false
			break
		}
	}

	if allPassed {
		e.logger.Info().
			Str("service", service.Name).
			Int("probe_count", len(results)).
			Msg("All health probes passed")
	} else {
		e.logger.Warn().
			Str("service", service.Name).
			Int("probe_count", len(results)).
			Msg("Some health probes failed")
	}

	return results
}

// AllProbesPassed checks if all probes in the results passed
func AllProbesPassed(results []state.ProbeResult) bool {
	if len(results) == 0 {
		return true // No probes = pass
	}

	for _, result := range results {
		if !result.Success {
			return false
		}
	}

	return true
}
