package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

// ComposeExecutor handles updates for Docker Compose projects
type ComposeExecutor struct {
	runner       *docker.ComposeRunner
	dockerClient *docker.Client
	logger       *logging.Logger
}

// NewComposeExecutor creates a new compose executor
func NewComposeExecutor(dockerClient *docker.Client, logger *logging.Logger) *ComposeExecutor {
	return &ComposeExecutor{
		runner:       docker.NewComposeRunner(),
		dockerClient: dockerClient,
		logger:       logger.WithComponent("compose-executor"),
	}
}

// UpdateService updates a service in a compose project
func (e *ComposeExecutor) UpdateService(ctx context.Context, target *state.Target, service *state.Service) error {
	e.logger.Info().
		Str("target", target.Name).
		Str("service", service.Name).
		Str("image", service.Image).
		Msg("Updating compose service")

	// Step 1: Pull the latest image
	e.logger.Info().
		Str("service", service.Name).
		Msg("Pulling latest image")

	pullStart := time.Now()
	if err := e.runner.Pull(ctx, target.Path, service.Name); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	pullDuration := time.Since(pullStart)

	e.logger.Info().
		Str("service", service.Name).
		Dur("duration", pullDuration).
		Msg("Image pull completed")

	// Step 2: Recreate service with new image (force recreate to pick up new digest)
	e.logger.Info().
		Str("service", service.Name).
		Msg("Recreating service")

	upStart := time.Now()
	if err := e.runner.Up(ctx, target.Path, service.Name, true); err != nil {
		return fmt.Errorf("failed to recreate service: %w", err)
	}
	upDuration := time.Since(upStart)

	e.logger.Info().
		Str("service", service.Name).
		Dur("duration", upDuration).
		Msg("Service recreated successfully")

	return nil
}

// GetNewDigest gets the digest of the currently running container after update
func (e *ComposeExecutor) GetNewDigest(ctx context.Context, target *state.Target, service *state.Service) (string, error) {
	// List containers to find the updated service
	containers, err := e.dockerClient.ListContainers(ctx, false)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// Find container for this service
	for _, container := range containers {
		// Check if it's part of the compose project and matches the service name
		if container.Labels["com.docker.compose.project"] == target.Name &&
			container.Labels["com.docker.compose.service"] == service.Name {

			// Inspect to get image digest
			inspect, err := e.dockerClient.InspectContainer(ctx, container.ID)
			if err != nil {
				return "", fmt.Errorf("failed to inspect container: %w", err)
			}

			e.logger.Debug().
				Str("service", service.Name).
				Str("digest", inspect.Image).
				Msg("Got new digest")

			return inspect.Image, nil
		}
	}

	return "", fmt.Errorf("container not found for service %s", service.Name)
}

// Rollback rolls back a service to a specific digest
func (e *ComposeExecutor) Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error {
	e.logger.Warn().
		Str("target", target.Name).
		Str("service", service.Name).
		Str("digest", digest).
		Msg("Rolling back service to previous digest")

	// Step 1: Pull the specific digest
	// Format: image@digest
	imageWithDigest := fmt.Sprintf("%s@%s", service.Image, digest)

	e.logger.Info().
		Str("image", imageWithDigest).
		Msg("Pulling previous digest")

	if err := e.dockerClient.ImagePull(ctx, imageWithDigest); err != nil {
		return fmt.Errorf("failed to pull previous digest: %w", err)
	}

	// Step 2: Tag with original tag for compose compatibility
	taggedImage := service.Image
	if strings.Contains(taggedImage, "@") {
		// Remove digest, keep tag/repository
		parts := strings.Split(service.Image, "@")
		taggedImage = parts[0]
	}

	e.logger.Debug().
		Str("source", digest).
		Str("target", taggedImage).
		Msg("Tagging image")

	if err := e.dockerClient.ImageTag(ctx, digest, taggedImage); err != nil {
		e.logger.Warn().Err(err).Msg("Failed to tag image, continuing anyway")
	}

	// Step 3: Recreate service with rolled-back image
	e.logger.Info().
		Str("service", service.Name).
		Msg("Recreating service with previous version")

	if err := e.runner.Up(ctx, target.Path, service.Name, true); err != nil {
		return fmt.Errorf("failed to recreate service during rollback: %w", err)
	}

	e.logger.Info().
		Str("service", service.Name).
		Msg("Rollback completed successfully")

	return nil
}
