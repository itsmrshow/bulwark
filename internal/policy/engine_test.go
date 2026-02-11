package policy

import (
	"context"
	"testing"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

func TestShouldRollbackReturnsFalseWhenAlreadyRolledBack(t *testing.T) {
	engine := NewEngine(logging.Default())

	result := &state.UpdateResult{
		Success:           false,
		RollbackPerformed: true,
	}

	if engine.ShouldRollback(context.Background(), result) {
		t.Fatalf("expected false when rollback already performed")
	}
}

func TestShouldRollbackReturnsTrueForFailedUpdateWithoutRollback(t *testing.T) {
	engine := NewEngine(logging.Default())

	result := &state.UpdateResult{
		Success:           false,
		RollbackPerformed: false,
	}

	if !engine.ShouldRollback(context.Background(), result) {
		t.Fatalf("expected true for failed update without rollback")
	}
}
