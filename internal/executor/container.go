package executor

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// ContainerExecutor handles updates for loose containers via safe definitions.
type ContainerExecutor struct {
	composeExec composeUpdater
	logger      *logging.Logger
}

// NewContainerExecutor creates a new container executor.
func NewContainerExecutor(composeExec composeUpdater, logger *logging.Logger) *ContainerExecutor {
	return &ContainerExecutor{
		composeExec: composeExec,
		logger:      logger.WithComponent("container-executor"),
	}
}

// UpdateService updates a loose container by delegating to a compose definition.
func (e *ContainerExecutor) UpdateService(ctx context.Context, target *state.Target, service *state.Service) error {
	if service == nil || target == nil {
		return NewSkipError("missing target or service")
	}

	if !service.Labels.Enabled {
		return NewSkipError("bulwark.enabled is not true")
	}

	definition, err := ParseDefinition(service.Labels.Definition)
	if err != nil {
		return NewSkipError(fmt.Sprintf("invalid definition: %v", err))
	}

	composeTarget := &state.Target{
		Type: state.TargetTypeCompose,
		Name: composeProjectName(definition.ComposePath),
		Path: definition.ComposePath,
	}
	composeService := &state.Service{
		Name:  definition.Service,
		Image: service.Image,
	}

	e.logger.Info().
		Str("container", service.Name).
		Str("compose_path", definition.ComposePath).
		Str("compose_service", definition.Service).
		Msg("Delegating update to compose executor")

	return e.composeExec.UpdateService(ctx, composeTarget, composeService)
}

// Rollback rolls back a loose container by delegating to a compose definition.
func (e *ContainerExecutor) Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error {
	if service == nil || target == nil {
		return NewSkipError("missing target or service")
	}

	if !service.Labels.Enabled {
		return NewSkipError("bulwark.enabled is not true")
	}

	definition, err := ParseDefinition(service.Labels.Definition)
	if err != nil {
		return NewSkipError(fmt.Sprintf("invalid definition: %v", err))
	}

	composeTarget := &state.Target{
		Type: state.TargetTypeCompose,
		Name: composeProjectName(definition.ComposePath),
		Path: definition.ComposePath,
	}
	composeService := &state.Service{
		Name:  definition.Service,
		Image: service.Image,
	}

	e.logger.Warn().
		Str("container", service.Name).
		Str("compose_path", definition.ComposePath).
		Str("compose_service", definition.Service).
		Msg("Delegating rollback to compose executor")

	return e.composeExec.Rollback(ctx, composeTarget, composeService, digest)
}

func composeProjectName(composePath string) string {
	if composePath == "" {
		return "unknown"
	}
	return filepath.Base(filepath.Dir(composePath))
}
