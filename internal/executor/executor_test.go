package executor

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

type fakeComposeUpdater struct {
	updateCalled    int
	getDigestCalled int
	rollbackCalled  int
	updateErr       error
	digest          string
}

func (f *fakeComposeUpdater) UpdateService(ctx context.Context, target *state.Target, service *state.Service) error {
	f.updateCalled++
	return f.updateErr
}

func (f *fakeComposeUpdater) GetNewDigest(ctx context.Context, target *state.Target, service *state.Service) (string, error) {
	f.getDigestCalled++
	if f.digest == "" {
		f.digest = "sha256:updated"
	}
	return f.digest, nil
}

func (f *fakeComposeUpdater) Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error {
	f.rollbackCalled++
	return nil
}

type fakeContainerUpdater struct {
	updateCalled   int
	rollbackCalled int
	updateErr      error
}

func (f *fakeContainerUpdater) UpdateService(ctx context.Context, target *state.Target, service *state.Service) error {
	f.updateCalled++
	return f.updateErr
}

func (f *fakeContainerUpdater) Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error {
	f.rollbackCalled++
	return nil
}

type fakeLockManager struct {
	lockCalled   int
	unlockCalled int
	lastTargetID string
	lockErr      error
}

func (f *fakeLockManager) Lock(ctx context.Context, targetID string, timeout time.Duration) error {
	f.lockCalled++
	f.lastTargetID = targetID
	return f.lockErr
}

func (f *fakeLockManager) Unlock(targetID string) {
	f.unlockCalled++
}

func TestExecutorUsesComposeUpdaterForComposeTargets(t *testing.T) {
	compose := &fakeComposeUpdater{}
	container := &fakeContainerUpdater{}
	locks := &fakeLockManager{}

	exec := &Executor{
		composeExec:   compose,
		containerExec: container,
		lockManager:   locks,
		logger:        logging.Default(),
		dryRun:        false,
	}

	target := &state.Target{
		ID:   "compose-1",
		Type: state.TargetTypeCompose,
		Name: "app",
		Path: "/tmp/compose.yml",
	}
	service := &state.Service{
		ID:            "svc-1",
		Name:          "web",
		CurrentDigest: "sha256:old",
		Image:         "nginx:latest",
		Labels:        state.DefaultLabels(),
	}

	result := exec.ExecuteUpdate(context.Background(), target, service, "sha256:new")

	if !result.Success {
		t.Fatalf("expected success, got error %v", result.Error)
	}
	if compose.updateCalled != 1 {
		t.Fatalf("expected compose updater called once, got %d", compose.updateCalled)
	}
	if container.updateCalled != 0 {
		t.Fatalf("expected container updater not called, got %d", container.updateCalled)
	}
	if locks.lockCalled != 1 || locks.unlockCalled != 1 {
		t.Fatalf("expected lock/unlock called once, got lock=%d unlock=%d", locks.lockCalled, locks.unlockCalled)
	}
	if compose.getDigestCalled != 1 {
		t.Fatalf("expected GetNewDigest called once, got %d", compose.getDigestCalled)
	}
}

func TestExecutorUsesContainerUpdaterForContainerTargets(t *testing.T) {
	compose := &fakeComposeUpdater{}
	container := &fakeContainerUpdater{}
	locks := &fakeLockManager{}

	exec := &Executor{
		composeExec:   compose,
		containerExec: container,
		lockManager:   locks,
		logger:        logging.Default(),
		dryRun:        false,
	}

	target := &state.Target{
		ID:   "container-1",
		Type: state.TargetTypeContainer,
		Name: "loose",
		Path: "container-id",
	}
	service := &state.Service{
		ID:            "svc-1",
		Name:          "loose",
		CurrentDigest: "sha256:old",
		Image:         "nginx:latest",
		Labels:        state.DefaultLabels(),
	}

	result := exec.ExecuteUpdate(context.Background(), target, service, "sha256:new")

	if !result.Success {
		t.Fatalf("expected success, got error %v", result.Error)
	}
	if container.updateCalled != 1 {
		t.Fatalf("expected container updater called once, got %d", container.updateCalled)
	}
	if compose.updateCalled != 0 {
		t.Fatalf("expected compose updater not called, got %d", compose.updateCalled)
	}
	if locks.lockCalled != 1 || locks.unlockCalled != 1 {
		t.Fatalf("expected lock/unlock called once, got lock=%d unlock=%d", locks.lockCalled, locks.unlockCalled)
	}
	if compose.getDigestCalled != 0 {
		t.Fatalf("expected GetNewDigest not called, got %d", compose.getDigestCalled)
	}
}
