package state

import (
	"context"
	"time"
)

// Store defines the interface for state persistence
type Store interface {
	// Initialize the store (create tables, run migrations)
	Initialize(ctx context.Context) error

	// Close the store connection
	Close() error

	// Target operations
	SaveTarget(ctx context.Context, target *Target) error
	GetTarget(ctx context.Context, id string) (*Target, error)
	GetTargetByName(ctx context.Context, name string) (*Target, error)
	ListTargets(ctx context.Context) ([]Target, error)
	DeleteTarget(ctx context.Context, id string) error

	// Service operations
	SaveService(ctx context.Context, service *Service) error
	GetService(ctx context.Context, id string) (*Service, error)
	GetServicesByTarget(ctx context.Context, targetID string) ([]Service, error)
	DeleteService(ctx context.Context, id string) error

	// Update history operations
	SaveUpdateResult(ctx context.Context, result *UpdateResult) error
	GetUpdateHistory(ctx context.Context, limit int) ([]UpdateResult, error)
	GetUpdateHistoryByTarget(ctx context.Context, targetID string, limit int) ([]UpdateResult, error)
	GetUpdateHistoryByService(ctx context.Context, serviceID string, limit int) ([]UpdateResult, error)
	GetLastSuccessfulUpdate(ctx context.Context, serviceID string) (*UpdateResult, error)

	// Cleanup operations
	PruneHistory(ctx context.Context, olderThan time.Time) error
	PruneStaleTargets(ctx context.Context, olderThan time.Time) error

	// Settings operations
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}
