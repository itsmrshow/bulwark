package executor

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

type fakeComposeExecutor struct {
	updateCalled int
	lastTarget   *state.Target
	lastService  *state.Service
	updateErr    error
}

func (f *fakeComposeExecutor) UpdateService(ctx context.Context, target *state.Target, service *state.Service) error {
	f.updateCalled++
	f.lastTarget = target
	f.lastService = service
	return f.updateErr
}

func (f *fakeComposeExecutor) GetNewDigest(ctx context.Context, target *state.Target, service *state.Service) (string, error) {
	return "", nil
}

func (f *fakeComposeExecutor) Rollback(ctx context.Context, target *state.Target, service *state.Service, digest string) error {
	return nil
}

func TestContainerExecutorDelegatesWithValidDefinition(t *testing.T) {
	tmp, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())

	definition := fmt.Sprintf("compose:%s#service=web", tmp.Name())
	service := &state.Service{
		Name:  "loose-container",
		Image: "nginx:latest",
		Labels: state.Labels{
			Enabled:    true,
			Definition: definition,
		},
	}

	target := &state.Target{
		ID:   "container-1",
		Type: state.TargetTypeContainer,
		Name: "loose-container",
	}

	fake := &fakeComposeExecutor{}
	exec := NewContainerExecutor(fake, logging.Default())

	if err := exec.UpdateService(context.Background(), target, service); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if fake.updateCalled != 1 {
		t.Fatalf("expected compose update to be called once, got %d", fake.updateCalled)
	}
	if fake.lastTarget == nil || fake.lastTarget.Path != tmp.Name() {
		t.Fatalf("expected compose path %q, got %#v", tmp.Name(), fake.lastTarget)
	}
	if fake.lastService == nil || fake.lastService.Name != "web" {
		t.Fatalf("expected compose service %q, got %#v", "web", fake.lastService)
	}
}

func TestContainerExecutorSkipsWithoutDefinition(t *testing.T) {
	service := &state.Service{
		Name:  "loose-container",
		Image: "nginx:latest",
		Labels: state.Labels{
			Enabled: true,
		},
	}

	target := &state.Target{
		ID:   "container-1",
		Type: state.TargetTypeContainer,
		Name: "loose-container",
	}

	fake := &fakeComposeExecutor{}
	exec := NewContainerExecutor(fake, logging.Default())

	err := exec.UpdateService(context.Background(), target, service)
	if err == nil || !IsSkipError(err) {
		t.Fatalf("expected skip error, got %v", err)
	}
	if fake.updateCalled != 0 {
		t.Fatalf("expected compose update not to be called, got %d", fake.updateCalled)
	}
}

func TestContainerExecutorSkipsWhenDisabled(t *testing.T) {
	service := &state.Service{
		Name:  "loose-container",
		Image: "nginx:latest",
		Labels: state.Labels{
			Enabled:    false,
			Definition: "compose:/abs/path/compose.yml#service=web",
		},
	}

	target := &state.Target{
		ID:   "container-1",
		Type: state.TargetTypeContainer,
		Name: "loose-container",
	}

	fake := &fakeComposeExecutor{}
	exec := NewContainerExecutor(fake, logging.Default())

	err := exec.UpdateService(context.Background(), target, service)
	if err == nil || !IsSkipError(err) {
		t.Fatalf("expected skip error, got %v", err)
	}
	if fake.updateCalled != 0 {
		t.Fatalf("expected compose update not to be called, got %d", fake.updateCalled)
	}
}
