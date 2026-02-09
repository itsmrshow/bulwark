package executor

import (
	"context"
	"time"

	"github.com/itsmrshow/bulwark/internal/state"
)

type composeUpdater interface {
	UpdateService(ctx context.Context, target *state.Target, service *state.Service) error
	GetNewDigest(ctx context.Context, target *state.Target, service *state.Service) (string, error)
	Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error
}

type containerUpdater interface {
	UpdateService(ctx context.Context, target *state.Target, service *state.Service) error
	Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error
}

type lockManager interface {
	Lock(ctx context.Context, targetID string, timeout time.Duration) error
	Unlock(targetID string)
}
