package probe

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

// LogProbe checks container logs for a regex pattern
type LogProbe struct {
	dockerClient *docker.Client
	containerID  string
	pattern      *regexp.Regexp
	windowSec    int
	config       Config
	logger       *logging.Logger
}

// NewLogProbe creates a new log probe.
// Returns a probe whose Execute will return failure if the pattern is invalid.
func NewLogProbe(dockerClient *docker.Client, containerID, pattern string, windowSec int, config Config, logger *logging.Logger) *LogProbe {
	if windowSec <= 0 {
		windowSec = 30
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		// Store nil pattern; Execute will report a clear failure.
		logger.WithComponent("log-probe").Warn().
			Str("pattern", pattern).
			Err(err).
			Msg("Invalid log probe regex pattern")
		compiled = nil
	}

	return &LogProbe{
		dockerClient: dockerClient,
		containerID:  containerID,
		pattern:      compiled,
		windowSec:    windowSec,
		config:       config,
		logger:       logger.WithComponent("log-probe"),
	}
}

// Type returns the probe type
func (p *LogProbe) Type() state.ProbeType {
	return state.ProbeTypeLog
}

// Execute runs the log probe
func (p *LogProbe) Execute(ctx context.Context) *state.ProbeResult {
	if p.pattern == nil {
		return &state.ProbeResult{
			Type:    p.Type(),
			Success: false,
			Message: "invalid regex pattern",
		}
	}

	p.logger.Debug().
		Str("container_id", p.containerID[:min(12, len(p.containerID))]).
		Str("pattern", p.pattern.String()).
		Int("window_sec", p.windowSec).
		Msg("Starting log probe")

	success, duration, message := executeWithRetries(ctx, p.config, func(ctx context.Context) error {
		return p.checkLogs(ctx)
	})

	result := &state.ProbeResult{
		Type:     p.Type(),
		Success:  success,
		Duration: duration,
		Message:  message,
	}

	if success {
		p.logger.Info().
			Str("pattern", p.pattern.String()).
			Dur("duration", duration).
			Msg("Log probe succeeded")
	} else {
		p.logger.Warn().
			Str("pattern", p.pattern.String()).
			Str("error", message).
			Msg("Log probe failed")
	}

	return result
}

// checkLogs reads recent container logs and checks for the pattern
func (p *LogProbe) checkLogs(ctx context.Context) error {
	since := time.Now().Add(-time.Duration(p.windowSec) * time.Second)

	reader, err := p.dockerClient.ContainerLogsSince(ctx, p.containerID, since, "500")
	if err != nil {
		return fmt.Errorf("failed to fetch container logs: %w", err)
	}
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		// Docker multiplexed log streams have an 8-byte header per frame.
		// Strip it if present (header byte 0 is stream type: 1=stdout, 2=stderr).
		if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
			line = line[8:]
		}
		if p.pattern.Match(line) {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return fmt.Errorf("pattern %q not found in container logs", p.pattern.String())
}
