package probe

import (
	"context"
	"fmt"

	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// DockerProbe checks Docker's built-in HEALTHCHECK status
type DockerProbe struct {
	dockerClient *docker.Client
	containerID  string
	config       Config
	logger       *logging.Logger
}

// NewDockerProbe creates a new Docker HEALTHCHECK probe
func NewDockerProbe(dockerClient *docker.Client, containerID string, config Config, logger *logging.Logger) *DockerProbe {
	return &DockerProbe{
		dockerClient: dockerClient,
		containerID:  containerID,
		config:       config,
		logger:       logger.WithComponent("docker-probe"),
	}
}

// Type returns the probe type
func (p *DockerProbe) Type() state.ProbeType {
	return state.ProbeTypeDocker
}

// Execute runs the Docker HEALTHCHECK probe
func (p *DockerProbe) Execute(ctx context.Context) *state.ProbeResult {
	p.logger.Debug().
		Str("container_id", p.containerID[:12]).
		Msg("Starting Docker HEALTHCHECK probe")

	success, duration, message := executeWithRetries(ctx, p.config, func(ctx context.Context) error {
		return p.checkHealth(ctx)
	})

	result := &state.ProbeResult{
		Type:     p.Type(),
		Success:  success,
		Duration: duration,
		Message:  message,
	}

	if success {
		p.logger.Info().
			Str("container_id", p.containerID[:12]).
			Dur("duration", duration).
			Msg("Docker HEALTHCHECK probe succeeded")
	} else {
		p.logger.Warn().
			Str("container_id", p.containerID[:12]).
			Str("error", message).
			Msg("Docker HEALTHCHECK probe failed")
	}

	return result
}

// checkHealth checks the Docker container health status
func (p *DockerProbe) checkHealth(ctx context.Context) error {
	inspect, err := p.dockerClient.InspectContainer(ctx, p.containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Check if container is running
	if !inspect.State.Running {
		return fmt.Errorf("container is not running (status: %s)", inspect.State.Status)
	}

	// Check health status if available
	if inspect.State.Health != nil {
		healthStatus := inspect.State.Health.Status

		switch healthStatus {
		case "healthy":
			return nil
		case "unhealthy":
			return fmt.Errorf("container health status is unhealthy")
		case "starting":
			return fmt.Errorf("container health is still starting")
		default:
			return fmt.Errorf("unknown health status: %s", healthStatus)
		}
	}

	// No health check configured, just verify it's running
	return nil
}
