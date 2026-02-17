package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/metrics"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/probe"
	"github.com/itsmrshow/bulwark/internal/state"
)

// Executor orchestrates updates for all target types
type Executor struct {
	composeExec   composeUpdater
	containerExec containerUpdater
	lockManager   lockManager
	policyEngine  *policy.Engine
	probeEngine   *probe.Engine
	store         state.Store
	dockerClient  *docker.Client
	logger        *logging.Logger
	dryRun        bool
	lockTimeout   time.Duration
}

// NewExecutor creates a new executor
func NewExecutor(dockerClient *docker.Client, policyEngine *policy.Engine, store state.Store, logger *logging.Logger, dryRun bool) *Executor {
	composeExec := NewComposeExecutor(dockerClient, logger)
	return &Executor{
		composeExec:   composeExec,
		containerExec: NewContainerExecutor(composeExec, logger),
		lockManager:   NewLockManager(logger),
		policyEngine:  policyEngine,
		probeEngine:   probe.NewEngine(dockerClient, logger),
		store:         store,
		dockerClient:  dockerClient,
		logger:        logger.WithComponent("executor"),
		dryRun:        dryRun,
		lockTimeout:   5 * time.Minute,
	}
}

// WithLockTimeout sets the lock acquisition timeout
func (e *Executor) WithLockTimeout(d time.Duration) *Executor {
	e.lockTimeout = d
	return e
}

// ExecuteUpdate performs an update for a service
func (e *Executor) ExecuteUpdate(ctx context.Context, target *state.Target, service *state.Service, newDigest string) *state.UpdateResult {
	result := &state.UpdateResult{
		TargetID:          target.ID,
		ServiceID:         service.ID,
		ServiceName:       service.Name,
		OldDigest:         service.CurrentDigest,
		NewDigest:         newDigest,
		Success:           false,
		RollbackPerformed: false,
		ProbeResults:      []state.ProbeResult{},
		StartedAt:         time.Now(),
	}

	e.logger.Info().
		Str("target", target.Name).
		Str("service", service.Name).
		Str("old_digest", service.CurrentDigest[:min(12, len(service.CurrentDigest))]).
		Str("new_digest", newDigest[:min(12, len(newDigest))]).
		Msg("Starting update")

	// Check if dry-run
	if e.dryRun {
		e.logger.Info().Msg("DRY RUN: Would update service")
		result.Success = true
		result.CompletedAt = time.Now()
		return result
	}

	// Acquire lock
	if err := e.lockManager.Lock(ctx, target.ID, e.lockTimeout); err != nil {
		result.Error = fmt.Errorf("failed to acquire lock: %w", err)
		result.CompletedAt = time.Now()
		return result
	}
	defer e.lockManager.Unlock(target.ID)

	// Perform update based on target type
	var updateErr error
	switch target.Type {
	case state.TargetTypeCompose:
		updateErr = e.composeExec.UpdateService(ctx, target, service)
	case state.TargetTypeContainer:
		updateErr = e.containerExec.UpdateService(ctx, target, service)
	default:
		updateErr = fmt.Errorf("unknown target type: %s", target.Type)
	}

	if updateErr != nil {
		result.Error = updateErr
		result.Success = false
		if IsSkipError(updateErr) {
			result.NewDigest = result.OldDigest
		}
		result.CompletedAt = time.Now()

		if !IsSkipError(updateErr) {
			metrics.UpdatesTotal.WithLabelValues(target.Name, service.Name, "failed").Inc()
		}

		if IsSkipError(updateErr) {
			e.logger.Info().
				Err(updateErr).
				Str("service", service.Name).
				Msg("Update skipped")
		} else {
			e.logger.Error().
				Err(updateErr).
				Str("service", service.Name).
				Msg("Update failed")
		}

		return result
	}

	// Get new digest after update
	var actualNewDigest string
	switch target.Type {
	case state.TargetTypeCompose:
		var err error
		actualNewDigest, err = e.composeExec.GetNewDigest(ctx, target, service)
		if err != nil {
			e.logger.Warn().Err(err).Msg("Failed to get new digest, using expected digest")
			actualNewDigest = newDigest
		}
	case state.TargetTypeContainer:
		// Best-effort only; we don't reconstruct container state for loose containers
		actualNewDigest = newDigest
	default:
		actualNewDigest = newDigest
	}
	result.NewDigest = actualNewDigest

	// Run health probes if configured (skip for dry-run and if probe type is none)
	if service.Labels.Probe.Type != state.ProbeTypeNone {
		e.logger.Info().
			Str("service", service.Name).
			Str("probe_type", string(service.Labels.Probe.Type)).
			Msg("Running health probes")

		// Find the container ID for probe execution
		containerID, err := e.findContainerID(ctx, target, service)
		if err != nil {
			e.logger.Warn().Err(err).Msg("Failed to find container ID for probes, skipping")
		} else {
			// Execute probes
			probeResults := e.probeEngine.ExecuteProbes(ctx, target, service, containerID)
			result.ProbeResults = probeResults

			// Check if all probes passed
			if !probe.AllProbesPassed(probeResults) {
				metrics.UpdatesTotal.WithLabelValues(target.Name, service.Name, "rolled_back").Inc()

				e.logger.Error().
					Str("service", service.Name).
					Msg("Health probes failed, initiating rollback")

				// Perform rollback
				rollbackErr := e.ExecuteRollback(ctx, target, service, result)
				if rollbackErr != nil {
					result.Error = fmt.Errorf("update succeeded but probes failed, rollback also failed: %w", rollbackErr)
				} else {
					result.Error = fmt.Errorf("update succeeded but health probes failed, rolled back to previous version")
				}

				result.Success = false
				result.CompletedAt = time.Now()

				// Save failed result
				if e.store != nil {
					if err := e.store.SaveUpdateResult(ctx, result); err != nil {
						e.logger.Warn().Err(err).Msg("Failed to save update result to store")
					}
				}

				return result
			}

			e.logger.Info().
				Str("service", service.Name).
				Int("probe_count", len(probeResults)).
				Msg("All health probes passed")
		}
	}

	// Update successful
	result.Success = true
	result.CompletedAt = time.Now()

	metrics.UpdatesTotal.WithLabelValues(target.Name, service.Name, "success").Inc()

	e.logger.Info().
		Str("service", service.Name).
		Dur("duration", result.CompletedAt.Sub(result.StartedAt)).
		Msg("Update completed successfully")

	// Save result to store if available
	if e.store != nil {
		if err := e.store.SaveUpdateResult(ctx, result); err != nil {
			e.logger.Warn().Err(err).Msg("Failed to save update result to store")
		}
	}

	return result
}

// ExecuteRollback rolls back a failed update
func (e *Executor) ExecuteRollback(ctx context.Context, target *state.Target, service *state.Service, result *state.UpdateResult) error {
	e.logger.Warn().
		Str("target", target.Name).
		Str("service", service.Name).
		Msg("Executing rollback")

	if e.dryRun {
		e.logger.Info().Msg("DRY RUN: Would rollback service")
		return nil
	}

	// Rollback based on target type
	var rollbackErr error
	switch target.Type {
	case state.TargetTypeCompose:
		rollbackErr = e.composeExec.Rollback(ctx, target, service, result.OldDigest)
	case state.TargetTypeContainer:
		rollbackErr = e.containerExec.Rollback(ctx, target, service, result.OldDigest)
	default:
		rollbackErr = fmt.Errorf("unknown target type: %s", target.Type)
	}

	if rollbackErr != nil {
		e.logger.Error().
			Err(rollbackErr).
			Str("service", service.Name).
			Msg("Rollback failed")
		return rollbackErr
	}

	// Update result to reflect rollback
	result.RollbackPerformed = true
	result.RollbackDigest = result.OldDigest

	metrics.RollbacksTotal.WithLabelValues(target.Name, service.Name).Inc()

	e.logger.Info().
		Str("service", service.Name).
		Msg("Rollback completed successfully")

	return nil
}

// findContainerID finds the container ID for a service
func (e *Executor) findContainerID(ctx context.Context, target *state.Target, service *state.Service) (string, error) {
	containers, err := e.dockerClient.ListContainers(ctx, false)
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// For compose targets, match by project and service labels
	if target.Type == state.TargetTypeCompose {
		for _, container := range containers {
			if container.Labels["com.docker.compose.project"] == target.Name &&
				container.Labels["com.docker.compose.service"] == service.Name {
				return container.ID, nil
			}
		}
	}

	// For container targets, use the path (which is the container ID)
	if target.Type == state.TargetTypeContainer {
		return target.Path, nil
	}

	return "", fmt.Errorf("container not found for service %s", service.Name)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
