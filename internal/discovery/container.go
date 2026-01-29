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

// ScanContainers scans ALL running containers for bulwark labels
func (s *ContainerScanner) ScanContainers(ctx context.Context) ([]state.Target, error) {
	s.logger.Info().Msg("Scanning running containers for Bulwark labels")

	// List all running containers
	containers, err := s.dockerClient.ListContainers(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	s.logger.Debug().Int("total", len(containers)).Msg("Found containers")

	// Group containers by compose project, loose containers go into individual targets
	composeProjects := make(map[string][]docker.Container)
	var looseContainers []docker.Container

	for _, container := range containers {
		// Parse labels to check if bulwark is enabled
		labels := ParseLabels(container.Labels, container.Image)
		if !labels.Enabled {
			continue
		}

		// Check if part of a compose project
		if projectName, ok := container.Labels["com.docker.compose.project"]; ok {
			composeProjects[projectName] = append(composeProjects[projectName], container)
		} else {
			looseContainers = append(looseContainers, container)
		}
	}

	var targets []state.Target

	// Process compose projects
	for projectName, projectContainers := range composeProjects {
		target := s.createComposeTarget(ctx, projectName, projectContainers)
		if target != nil {
			targets = append(targets, *target)
		}
	}

	// Process loose containers
	for _, container := range looseContainers {
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
		containerName := getContainerName(container.Names)
		target := state.Target{
			ID:        state.GenerateTargetID(state.TargetTypeContainer, containerName, container.ID),
			Type:      state.TargetTypeContainer,
			Name:      containerName,
			Path:      container.ID, // Store container ID as path for loose containers
			Services:  []state.Service{},
			Labels:    labels,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create a service entry for the container
		service := state.Service{
			ID:            state.GenerateServiceID(target.ID, containerName),
			TargetID:      target.ID,
			Name:          containerName,
			Image:         container.Image,
			CurrentDigest: inspect.Image,
			Labels:        labels,
			HealthCheck:   parseContainerHealthCheck(inspect.State.Health),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
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

// createComposeTarget creates a target from a group of compose containers
func (s *ContainerScanner) createComposeTarget(ctx context.Context, projectName string, containers []docker.Container) *state.Target {
	if len(containers) == 0 {
		return nil
	}

	// Get the compose file path from the first container's working dir label (if available)
	composePath := ""
	if workingDir, ok := containers[0].Labels["com.docker.compose.project.working_dir"]; ok {
		composePath = workingDir + "/docker-compose.yml"
	}

	// Create target
	target := state.Target{
		ID:        state.GenerateTargetID(state.TargetTypeCompose, projectName, composePath),
		Type:      state.TargetTypeCompose,
		Name:      projectName,
		Path:      composePath,
		Services:  []state.Service{},
		Labels:    state.DefaultLabels(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add each container as a service
	for _, container := range containers {
		serviceName := container.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = getContainerName(container.Names)
		}

		// Parse labels from the running container
		labels := ParseLabels(container.Labels, container.Image)

		// Inspect container for more details
		inspect, err := s.dockerClient.InspectContainer(ctx, container.ID)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("container_id", container.ID).
				Msg("Failed to inspect container")
			continue
		}

		service := state.Service{
			ID:            state.GenerateServiceID(target.ID, serviceName),
			TargetID:      target.ID,
			Name:          serviceName,
			Image:         container.Image,
			CurrentDigest: inspect.Image,
			Labels:        labels,
			HealthCheck:   parseContainerHealthCheck(inspect.State.Health),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		target.Services = append(target.Services, service)

		s.logger.Debug().
			Str("project", projectName).
			Str("service", serviceName).
			Str("image", container.Image).
			Msg("Added compose service from container")
	}

	s.logger.Info().
		Str("project", projectName).
		Int("services", len(target.Services)).
		Msg("Found compose project from running containers")

	return &target
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
func parseContainerHealthCheck(health *docker.Health) *state.HealthCheck {
	// Docker's container inspect health is different from config health
	// We can't extract the full healthcheck config from runtime state
	// This is okay - we'll read it from the image or container config if needed
	return nil
}
