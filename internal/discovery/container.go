package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// ContainerScanner scans for loose containers with Bulwark labels
type ContainerScanner struct {
	logger       *logging.Logger
	dockerClient *docker.Client
}

// NewContainerScanner creates a new container scanner
func NewContainerScanner(logger *logging.Logger, dockerClient *docker.Client) *ContainerScanner {
	return &ContainerScanner{
		logger:       logger.WithComponent("container-scanner"),
		dockerClient: dockerClient,
	}
}

// ScanContainers scans for loose containers with bulwark.enabled=true
func (s *ContainerScanner) ScanContainers(ctx context.Context) ([]state.Target, error) {
	s.logger.Info().Msg("Scanning for labeled containers")

	// List all running containers
	containers, err := s.dockerClient.ListContainers(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	s.logger.Debug().Int("total", len(containers)).Msg("Found containers")

	var targets []state.Target

	for _, container := range containers {
		// Skip containers managed by compose (they're handled by ComposeScanner)
		if _, ok := container.Labels["com.docker.compose.project"]; ok {
			continue
		}

		// Parse labels
		labels := ParseLabels(container.Labels, container.Image)

		// Only include if bulwark.enabled=true
		if !labels.Enabled {
			continue
		}

		// Inspect container for more details
		inspect, err := s.dockerClient.InspectContainer(ctx, container.ID)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("container_id", container.ID).
				Msg("Failed to inspect container")
			continue
		}

		// Create target for this loose container
		target := state.Target{
			ID:        fmt.Sprintf("container_%s", container.ID[:12]),
			Type:      state.TargetTypeContainer,
			Name:      getContainerName(container.Names),
			Path:      "", // No path for loose containers
			Services:  []state.Service{},
			Labels:    labels,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create a service entry for the container
		service := state.Service{
			Name:          getContainerName(container.Names),
			Image:         container.Image,
			CurrentDigest: inspect.Image,
			Labels:        labels,
			HealthCheck:   parseContainerHealthCheck(inspect.State.Health),
		}

		target.Services = append(target.Services, service)

		targets = append(targets, target)

		s.logger.Info().
			Str("container", target.Name).
			Str("image", service.Image).
			Str("policy", string(labels.Policy)).
			Msg("Found enabled loose container")
	}

	return targets, nil
}

// getContainerName extracts a clean container name from the Names array
func getContainerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}

	// Docker returns names with a leading slash
	name := names[0]
	if len(name) > 0 && name[0] == '/' {
		return name[1:]
	}

	return name
}

// parseContainerHealthCheck converts Docker's Health to our format
func parseContainerHealthCheck(health *struct {
	Status        string
	FailingStreak int
	Log           []struct {
		Start    time.Time
		End      time.Time
		ExitCode int
		Output   string
	}
}) *state.HealthCheck {
	// Docker's container inspect health is different from config health
	// We can't extract the full healthcheck config from runtime state
	// This is okay - we'll read it from the image or container config if needed
	return nil
}
