package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yourusername/bulwark/internal/logging"
)

// LockManager manages per-target locks to prevent concurrent updates
type LockManager struct {
	locks  sync.Map // map[string]*sync.Mutex
	logger *logging.Logger
}

// NewLockManager creates a new lock manager
func NewLockManager(logger *logging.Logger) *LockManager {
	return &LockManager{
		logger: logger.WithComponent("lock-manager"),
	}
}

// Lock acquires a lock for the given target ID with timeout
func (lm *LockManager) Lock(ctx context.Context, targetID string, timeout time.Duration) error {
	lm.logger.Debug().
		Str("target_id", targetID).
		Dur("timeout", timeout).
		Msg("Attempting to acquire lock")

	// Get or create mutex for this target
	mutexInterface, _ := lm.locks.LoadOrStore(targetID, &sync.Mutex{})
	mutex := mutexInterface.(*sync.Mutex)

	// Try to acquire lock with timeout
	lockAcquired := make(chan struct{})
	go func() {
		mutex.Lock()
		close(lockAcquired)
	}()

	// Wait for lock or timeout
	select {
	case <-lockAcquired:
		lm.logger.Debug().
			Str("target_id", targetID).
			Msg("Lock acquired")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context canceled while waiting for lock: %w", ctx.Err())
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for lock on target %s", targetID)
	}
}

// Unlock releases the lock for the given target ID
func (lm *LockManager) Unlock(targetID string) {
	mutexInterface, ok := lm.locks.Load(targetID)
	if !ok {
		lm.logger.Warn().
			Str("target_id", targetID).
			Msg("Attempted to unlock non-existent lock")
		return
	}

	mutex := mutexInterface.(*sync.Mutex)
	mutex.Unlock()

	lm.logger.Debug().
		Str("target_id", targetID).
		Msg("Lock released")
}

// WithLock executes a function while holding the lock for a target
func (lm *LockManager) WithLock(ctx context.Context, targetID string, timeout time.Duration, fn func() error) error {
	if err := lm.Lock(ctx, targetID, timeout); err != nil {
		return err
	}
	defer lm.Unlock(targetID)

	return fn()
}
