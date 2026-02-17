package probe

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/metrics"
	"github.com/itsmrshow/bulwark/internal/state"
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

// ExecuteProbes runs all configured probes for a service.
// Probes run concurrently with fail-fast: the first failure cancels remaining probes.
func (e *Engine) ExecuteProbes(ctx context.Context, target *state.Target, service *state.Service, containerID string) []state.ProbeResult {
	probeConfig := service.Labels.Probe

	e.logger.Info().
		Str("service", service.Name).
		Str("probe_type", string(probeConfig.Type)).
		Msg("Executing health probes")

	// Collect probes and any immediate error results
	probes, errorResults := e.collectProbes(probeConfig, containerID)

	// If we only have error results (validation failures), return them
	if len(probes) == 0 {
		for _, result := range errorResults {
			e.logProbeResult(service.Name, result)
		}
		return errorResults
	}

	// Execute probes concurrently with fail-fast
	results := make([]state.ProbeResult, len(probes))
	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	firstFailure := false

	for i, p := range probes {
		g.Go(func() error {
			result := p.Execute(gctx)
			mu.Lock()
			results[i] = *result
			if !result.Success {
				firstFailure = true
			}
			mu.Unlock()
			if !result.Success {
				return fmt.Errorf("probe %s failed: %s", result.Type, result.Message)
			}
			return nil
		})
	}

	// Wait for all probes (errgroup cancels context on first error)
	_ = g.Wait()

	// Combine error results with probe results
	allResults := append(errorResults, results...)

	// Log detail for each result
	for _, result := range allResults {
		e.logProbeResult(service.Name, result)
	}

	// Log summary
	allPassed := !firstFailure && len(errorResults) == 0
	if allPassed {
		e.logger.Info().
			Str("service", service.Name).
			Int("probe_count", len(allResults)).
			Msg("All health probes passed")
	} else {
		e.logger.Warn().
			Str("service", service.Name).
			Int("probe_count", len(allResults)).
			Msg("Some health probes failed")
	}

	return allResults
}

// collectProbes builds probe instances and returns immediate error results for misconfigured probes.
func (e *Engine) collectProbes(probeConfig state.ProbeConfig, containerID string) ([]Probe, []state.ProbeResult) {
	var probes []Probe
	var errorResults []state.ProbeResult

	switch probeConfig.Type {
	case state.ProbeTypeDocker:
		probes = append(probes, NewDockerProbe(e.dockerClient, containerID, e.config, e.logger))

	case state.ProbeTypeHTTP:
		if probeConfig.HTTPUrl == "" {
			e.logger.Warn().Msg("HTTP probe configured but no URL provided, skipping")
			errorResults = append(errorResults, state.ProbeResult{
				Type:    state.ProbeTypeHTTP,
				Success: false,
				Message: "HTTP probe URL not configured",
			})
		} else {
			probes = append(probes, NewHTTPProbe(probeConfig.HTTPUrl, probeConfig.HTTPStatus, e.config, e.logger))
		}

	case state.ProbeTypeTCP:
		if probeConfig.TCPHost == "" || probeConfig.TCPPort == 0 {
			e.logger.Warn().Msg("TCP probe configured but host/port missing, skipping")
			errorResults = append(errorResults, state.ProbeResult{
				Type:    state.ProbeTypeTCP,
				Success: false,
				Message: "TCP probe host/port not configured",
			})
		} else {
			probes = append(probes, NewTCPProbe(probeConfig.TCPHost, probeConfig.TCPPort, e.config, e.logger))
		}

	case state.ProbeTypeStability:
		stabilityWindow := probeConfig.StabilitySec
		if stabilityWindow == 0 {
			stabilityWindow = 10
		}
		probes = append(probes, NewStabilityProbe(stabilityWindow, e.config, e.logger))

	case state.ProbeTypeLog:
		if probeConfig.LogPattern == "" {
			e.logger.Warn().Msg("Log probe configured but no pattern provided, skipping")
			errorResults = append(errorResults, state.ProbeResult{
				Type:    state.ProbeTypeLog,
				Success: false,
				Message: "Log probe pattern not configured",
			})
		} else {
			windowSec := probeConfig.WindowSec
			if windowSec == 0 {
				windowSec = 30
			}
			probes = append(probes, NewLogProbe(e.dockerClient, containerID, probeConfig.LogPattern, windowSec, e.config, e.logger))
		}

	case state.ProbeTypeNone:
		// No probe configured

	default:
		e.logger.Warn().
			Str("probe_type", string(probeConfig.Type)).
			Msg("Unknown probe type, skipping")
		errorResults = append(errorResults, state.ProbeResult{
			Type:    probeConfig.Type,
			Success: false,
			Message: fmt.Sprintf("unknown probe type: %s", probeConfig.Type),
		})
	}

	return probes, errorResults
}

// logProbeResult logs details of a single probe result and records metrics
func (e *Engine) logProbeResult(serviceName string, result state.ProbeResult) {
	resultLabel := "success"
	if !result.Success {
		resultLabel = "failure"
	}
	metrics.ProbesTotal.WithLabelValues(string(result.Type), resultLabel).Inc()
	if result.Duration > 0 {
		metrics.ProbeDuration.WithLabelValues(string(result.Type)).Observe(result.Duration.Seconds())
	}

	event := e.logger.Info()
	if !result.Success {
		event = e.logger.Warn()
	}
	event.
		Str("service", serviceName).
		Str("probe_type", string(result.Type)).
		Bool("success", result.Success).
		Dur("duration", result.Duration).
		Str("message", result.Message).
		Msg("Probe result")
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
