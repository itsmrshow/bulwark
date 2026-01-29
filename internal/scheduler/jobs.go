package scheduler

import (
	"context"
	"fmt"

	"github.com/yourusername/bulwark/internal/discovery"
	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/executor"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/policy"
	"github.com/yourusername/bulwark/internal/registry"
	"github.com/yourusername/bulwark/internal/state"
)

// CheckJob performs periodic update checks
type CheckJob struct {
	root           string
	dockerClient   *docker.Client
	registryClient *registry.Client
	discoverer     *discovery.Discoverer
	logger         *logging.Logger
}

// NewCheckJob creates a new check job
func NewCheckJob(root string, dockerClient *docker.Client, store state.Store, logger *logging.Logger) *CheckJob {
	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if store != nil {
		discoverer = discoverer.WithStore(store)
	}

	return &CheckJob{
		root:           root,
		dockerClient:   dockerClient,
		registryClient: registry.NewClient(logger),
		discoverer:     discoverer,
		logger:         logger.WithComponent("check-job"),
	}
}

// Name returns the job name
func (j *CheckJob) Name() string {
	return "check-updates"
}

// Execute runs the check job
func (j *CheckJob) Execute(ctx context.Context) error {
	j.logger.Info().Msg("Running scheduled update check")

	// Discover targets
	targets, err := j.discoverer.Discover(ctx, j.root)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Check for updates
	updateCount := 0
	for _, target := range targets {
		for _, service := range target.Services {
			if !service.Labels.Enabled {
				continue
			}

			remoteDigest, err := j.registryClient.FetchDigest(ctx, service.Image)
			if err != nil {
				j.logger.Warn().
					Err(err).
					Str("service", service.Name).
					Msg("Failed to fetch remote digest")
				continue
			}

			if registry.CompareDigests(service.CurrentDigest, remoteDigest) {
				updateCount++
				j.logger.Info().
					Str("target", target.Name).
					Str("service", service.Name).
					Str("image", service.Image).
					Msg("Update available")
			}
		}
	}

	j.logger.Info().
		Int("updates_available", updateCount).
		Msg("Update check completed")

	return nil
}

// ApplyJob performs periodic updates
type ApplyJob struct {
	root         string
	dockerClient *docker.Client
	executor     *executor.Executor
	discoverer   *discovery.Discoverer
	logger       *logging.Logger
}

// NewApplyJob creates a new apply job
func NewApplyJob(root string, dockerClient *docker.Client, policyEngine *policy.Engine, store state.Store, logger *logging.Logger) *ApplyJob {
	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if store != nil {
		discoverer = discoverer.WithStore(store)
	}

	return &ApplyJob{
		root:         root,
		dockerClient: dockerClient,
		executor:     executor.NewExecutor(dockerClient, policyEngine, store, logger, false),
		discoverer:   discoverer,
		logger:       logger.WithComponent("apply-job"),
	}
}

// Name returns the job name
func (j *ApplyJob) Name() string {
	return "apply-updates"
}

// Execute runs the apply job
func (j *ApplyJob) Execute(ctx context.Context) error {
	j.logger.Info().Msg("Running scheduled update apply")

	// Discover targets
	targets, err := j.discoverer.Discover(ctx, j.root)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	registryClient := registry.NewClient(j.logger)

	// Apply updates
	appliedCount := 0
	failedCount := 0

	for _, target := range targets {
		for _, service := range target.Services {
			if !service.Labels.Enabled {
				continue
			}

			// Fetch remote digest
			remoteDigest, err := registryClient.FetchDigest(ctx, service.Image)
			if err != nil {
				j.logger.Warn().
					Err(err).
					Str("service", service.Name).
					Msg("Failed to fetch remote digest")
				continue
			}

			// Check if update needed
			if !registry.CompareDigests(service.CurrentDigest, remoteDigest) {
				continue
			}

			// Apply update
			j.logger.Info().
				Str("target", target.Name).
				Str("service", service.Name).
				Msg("Applying update")

			result := j.executor.ExecuteUpdate(ctx, &target, &service, remoteDigest)

			if result.Success {
				appliedCount++
				j.logger.Info().
					Str("target", target.Name).
					Str("service", service.Name).
					Msg("Update applied successfully")
			} else {
				failedCount++
				j.logger.Error().
					Err(result.Error).
					Str("target", target.Name).
					Str("service", service.Name).
					Msg("Update failed")
			}
		}
	}

	j.logger.Info().
		Int("applied", appliedCount).
		Int("failed", failedCount).
		Msg("Scheduled update apply completed")

	if failedCount > 0 {
		return fmt.Errorf("%d updates failed", failedCount)
	}

	return nil
}
