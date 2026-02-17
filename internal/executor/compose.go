package executor

import (
	"context"
	"fmt"
	"os"
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
	if e.shouldSkipSelfUpdate(ctx, target, service) {
		e.logger.Warn().
			Str("target", target.Name).
			Str("service", service.Name).
			Msg("Skipping self-update for Bulwark service")
		return NewSkipError("self-update skipped: update Bulwark externally with 'docker compose pull bulwark && docker compose up -d bulwark'")
	}

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

func (e *ComposeExecutor) shouldSkipSelfUpdate(ctx context.Context, target *state.Target, service *state.Service) bool {
	if target == nil || service == nil {
		return false
	}

	if allowSelfUpdate(os.Getenv("BULWARK_ALLOW_SELF_UPDATE")) {
		return false
	}

	if e.dockerClient == nil {
		return false
	}

	selfContainerID := getSelfContainerID()
	if selfContainerID == "" {
		return false
	}

	inspect, err := e.dockerClient.InspectContainer(ctx, selfContainerID)
	if err != nil || inspect.Config == nil {
		return false
	}

	labels := inspect.Config.Labels
	if labels == nil {
		return false
	}

	selfProject := labels["com.docker.compose.project"]
	selfService := labels["com.docker.compose.service"]

	return isSameComposeService(target.Name, service.Name, selfProject, selfService)
}

func isSameComposeService(targetProject, targetService, selfProject, selfService string) bool {
	if targetProject == "" || targetService == "" || selfProject == "" || selfService == "" {
		return false
	}
	return targetProject == selfProject && targetService == selfService
}

func allowSelfUpdate(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "1" || value == "true" || value == "yes"
}

// getSelfContainerID determines the container ID of the running Bulwark instance.
// It tries multiple strategies in order of reliability:
// 1. BULWARK_CONTAINER_ID env var (explicit, most reliable)
// 2. /proc/1/cpuset (works in most Docker environments)
// 3. HOSTNAME env var (fallback, current behavior)
func getSelfContainerID() string {
	// Strategy 1: Explicit env var
	if id := strings.TrimSpace(os.Getenv("BULWARK_CONTAINER_ID")); id != "" {
		return id
	}

	// Strategy 2: Read from /proc/1/cpuset
	if data, err := os.ReadFile("/proc/1/cpuset"); err == nil {
		cpuset := strings.TrimSpace(string(data))
		// cpuset looks like /docker/<container_id> or /system.slice/docker-<id>.scope
		if parts := strings.Split(cpuset, "/"); len(parts) > 1 {
			last := parts[len(parts)-1]
			// Docker format: the last segment is the container ID
			if len(last) >= 12 && last != "" {
				// Handle systemd format: docker-<id>.scope
				last = strings.TrimPrefix(last, "docker-")
				last = strings.TrimSuffix(last, ".scope")
				if len(last) >= 12 {
					return last
				}
			}
		}
	}

	// Strategy 3: HOSTNAME (traditional fallback)
	return strings.TrimSpace(os.Getenv("HOSTNAME"))
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

	baseImage := service.Image
	if strings.Contains(baseImage, "@") {
		parts := strings.Split(service.Image, "@")
		baseImage = parts[0]
	}
	imageWithDigest := fmt.Sprintf("%s@%s", baseImage, digest)

	e.logger.Info().
		Str("image", imageWithDigest).
		Msg("Pulling previous digest")

	if err := e.dockerClient.ImagePull(ctx, imageWithDigest); err != nil {
		return fmt.Errorf("failed to pull previous digest: %w", err)
	}

	// Step 2: Pin rollback image via temporary compose override to guarantee digest recreation.
	overrideFile, err := os.CreateTemp("", "bulwark-rollback-*.yml")
	if err != nil {
		return fmt.Errorf("failed to create rollback override file: %w", err)
	}
	overridePath := overrideFile.Name()
	defer func() {
		_ = overrideFile.Close()
		_ = os.Remove(overridePath)
	}()

	overrideContent := fmt.Sprintf("services:\n  %s:\n    image: %s\n", service.Name, imageWithDigest)
	if _, err := overrideFile.WriteString(overrideContent); err != nil {
		return fmt.Errorf("failed to write rollback override file: %w", err)
	}

	// Step 3: Recreate service with rolled-back image
	e.logger.Info().
		Str("service", service.Name).
		Msg("Recreating service with previous version")

	if err := e.runner.UpWithOverride(ctx, target.Path, overridePath, service.Name, true); err != nil {
		return fmt.Errorf("failed to recreate service during rollback: %w", err)
	}

	e.logger.Info().
		Str("service", service.Name).
		Msg("Rollback completed successfully")

	return nil
}
