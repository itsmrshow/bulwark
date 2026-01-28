package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/policy"
	"github.com/yourusername/bulwark/internal/state"
)

// Executor orchestrates updates for all target types
type Executor struct {
	composeExec   composeUpdater
	containerExec containerUpdater
	lockManager   lockManager
	policyEngine  *policy.Engine
	store         state.Store
	logger        *logging.Logger
	dryRun        bool
}

// NewExecutor creates a new executor
func NewExecutor(dockerClient *docker.Client, policyEngine *policy.Engine, store state.Store, logger *logging.Logger, dryRun bool) *Executor {
	composeExec := NewComposeExecutor(dockerClient, logger)
	return &Executor{
		composeExec:   composeExec,
		containerExec: NewContainerExecutor(composeExec, logger),
		lockManager:   NewLockManager(logger),
		policyEngine:  policyEngine,
		store:         store,
		logger:        logger.WithComponent("executor"),
		dryRun:        dryRun,
	}
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

	// Acquire lock with 5-minute timeout
	lockTimeout := 5 * time.Minute
	if err := e.lockManager.Lock(ctx, target.ID, lockTimeout); err != nil {
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
	actualNewDigest := newDigest
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

	// Update successful
	result.Success = true
	result.CompletedAt = time.Now()

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

	// Save updated result
	if e.store != nil {
		if err := e.store.SaveUpdateResult(ctx, result); err != nil {
			e.logger.Warn().Err(err).Msg("Failed to save rollback result")
		}
	}

	e.logger.Info().
		Str("service", service.Name).
		Msg("Rollback completed successfully")

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
