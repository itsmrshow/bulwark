package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
)

func TestSQLiteStoreReusesIDsWhenTargetPathChanges(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "state.db")

	store, err := NewSQLiteStore(dbPath, logging.Default())
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Initialize(ctx); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	target1 := &Target{
		ID:     "target-old",
		Type:   TargetTypeCompose,
		Name:   "esp32",
		Path:   "/docker_data/esp32/docker-compose.yml",
		Labels: DefaultLabels(),
	}
	if err := store.SaveTarget(ctx, target1); err != nil {
		t.Fatalf("SaveTarget target1 failed: %v", err)
	}

	service1 := &Service{
		ID:            "service-old",
		TargetID:      target1.ID,
		Name:          "esphome",
		Image:         "ghcr.io/esphome/esphome:latest",
		CurrentDigest: "sha256:old",
		Labels:        DefaultLabels(),
	}
	if err := store.SaveService(ctx, service1); err != nil {
		t.Fatalf("SaveService service1 failed: %v", err)
	}

	// Same target name with a different discovered ID/path should reuse stored target ID.
	target2 := &Target{
		ID:     "target-new",
		Type:   TargetTypeCompose,
		Name:   "esp32",
		Path:   "/docker_data/esp32/docker-compose.yaml",
		Labels: DefaultLabels(),
	}
	if err := store.SaveTarget(ctx, target2); err != nil {
		t.Fatalf("SaveTarget target2 failed: %v", err)
	}
	if target2.ID != target1.ID {
		t.Fatalf("expected target ID to be reused; got %s want %s", target2.ID, target1.ID)
	}

	// Same service name on same target should reuse stored service ID.
	service2 := &Service{
		ID:            "service-new",
		TargetID:      target2.ID,
		Name:          "esphome",
		Image:         "ghcr.io/esphome/esphome:latest",
		CurrentDigest: "sha256:new",
		Labels:        DefaultLabels(),
	}
	if err := store.SaveService(ctx, service2); err != nil {
		t.Fatalf("SaveService service2 failed: %v", err)
	}
	if service2.ID != service1.ID {
		t.Fatalf("expected service ID to be reused; got %s want %s", service2.ID, service1.ID)
	}

	// History write should succeed with reused IDs (no FK failure).
	result := &UpdateResult{
		TargetID:          target2.ID,
		ServiceID:         service2.ID,
		ServiceName:       service2.Name,
		OldDigest:         "sha256:old",
		NewDigest:         "sha256:new",
		Success:           true,
		RollbackPerformed: false,
		ProbeResults:      []ProbeResult{},
		StartedAt:         time.Now().Add(-time.Second),
		CompletedAt:       time.Now(),
	}
	if err := store.SaveUpdateResult(ctx, result); err != nil {
		t.Fatalf("SaveUpdateResult failed: %v", err)
	}
}
