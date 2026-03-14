package planner

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/state"
)

type stubDiscoverer struct {
	targets []state.Target
}

func (s stubDiscoverer) Discover(ctx context.Context, basePath string) ([]state.Target, error) {
	return s.targets, nil
}

func (s stubDiscoverer) DiscoverTarget(ctx context.Context, basePath, targetID string) (*state.Target, error) {
	return &s.targets[0], nil
}

type stubRegistry struct {
	digest string
}

func (s stubRegistry) FetchDigest(ctx context.Context, image string) (string, error) {
	return s.digest, nil
}

type countingRegistry struct {
	mu    sync.Mutex
	calls map[string]int
}

func (c *countingRegistry) FetchDigest(ctx context.Context, image string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.calls == nil {
		c.calls = make(map[string]int)
	}
	c.calls[image]++
	return "sha256:new", nil
}

func TestPlannerBuildPlan(t *testing.T) {
	labels := state.DefaultLabels()
	labels.Enabled = true
	labels.Policy = state.PolicySafe
	labels.Tier = state.TierStateless
	labels.Probe = state.ProbeConfig{Type: state.ProbeTypeHTTP, HTTPUrl: "http://localhost", HTTPStatus: 200}

	targetID := state.GenerateTargetID(state.TargetTypeCompose, "demo", "/docker_data/demo/docker-compose.yml")
	serviceID := state.GenerateServiceID(targetID, "web")

	target := state.Target{
		ID:   targetID,
		Type: state.TargetTypeCompose,
		Name: "demo",
		Path: "/docker_data/demo/docker-compose.yml",
		Services: []state.Service{
			{
				ID:            serviceID,
				TargetID:      targetID,
				Name:          "web",
				Image:         "nginx:latest",
				CurrentDigest: "sha256:old",
				Labels:        labels,
			},
		},
	}

	plannerSvc := NewPlanner(
		logging.Default(),
		stubDiscoverer{targets: []state.Target{target}},
		stubRegistry{digest: "sha256:new"},
		policy.NewEngine(logging.Default()),
	)

	plan, err := plannerSvc.BuildPlan(context.Background(), PlanOptions{Root: "/docker_data"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plan.TargetCount != 1 {
		t.Fatalf("expected 1 target, got %d", plan.TargetCount)
	}
	if plan.ServiceCount != 1 {
		t.Fatalf("expected 1 service, got %d", plan.ServiceCount)
	}
	if plan.UpdateCount != 1 {
		t.Fatalf("expected 1 update, got %d", plan.UpdateCount)
	}
	if plan.AllowedCount != 1 {
		t.Fatalf("expected 1 allowed update, got %d", plan.AllowedCount)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(plan.Items))
	}
	item := plan.Items[0]
	if !item.UpdateAvailable {
		t.Fatalf("expected update available")
	}
	if item.Risk != RiskSafe {
		t.Fatalf("expected risk %s, got %s", RiskSafe, item.Risk)
	}
}

func TestMapHistoryClampsInvalidCompletionTimes(t *testing.T) {
	started := time.Date(2026, 2, 9, 20, 40, 49, 0, time.UTC)

	items := MapHistory([]state.UpdateResult{
		{
			TargetID:    "t1",
			ServiceID:   "s1",
			ServiceName: "svc",
			StartedAt:   started,
			CompletedAt: time.Time{},
		},
		{
			TargetID:    "t2",
			ServiceID:   "s2",
			ServiceName: "svc2",
			StartedAt:   started,
			CompletedAt: started.Add(-time.Second),
		},
	})

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	for _, item := range items {
		if item.DurationSec != 0 {
			t.Fatalf("expected zero duration for invalid completed_at, got %f", item.DurationSec)
		}
		if !item.CompletedAt.Equal(item.StartedAt) {
			t.Fatalf("expected completed_at to be clamped to started_at")
		}
	}
}

func TestPlannerBuildPlanDeduplicatesDigestFetchesByImage(t *testing.T) {
	labels := state.DefaultLabels()
	labels.Enabled = true
	labels.Probe = state.ProbeConfig{Type: state.ProbeTypeHTTP, HTTPUrl: "http://localhost", HTTPStatus: 200}

	targetID := state.GenerateTargetID(state.TargetTypeCompose, "demo", "/docker_data/demo/docker-compose.yml")
	target := state.Target{
		ID:   targetID,
		Type: state.TargetTypeCompose,
		Name: "demo",
		Path: "/docker_data/demo/docker-compose.yml",
		Services: []state.Service{
			{
				ID:            state.GenerateServiceID(targetID, "web-1"),
				TargetID:      targetID,
				Name:          "web-1",
				Image:         "nginx:latest",
				CurrentDigest: "sha256:old",
				Labels:        labels,
			},
			{
				ID:            state.GenerateServiceID(targetID, "web-2"),
				TargetID:      targetID,
				Name:          "web-2",
				Image:         "nginx:latest",
				CurrentDigest: "sha256:old",
				Labels:        labels,
			},
		},
	}

	registry := &countingRegistry{}
	plannerSvc := NewPlanner(
		logging.Default(),
		stubDiscoverer{targets: []state.Target{target}},
		registry,
		policy.NewEngine(logging.Default()),
	)

	if _, err := plannerSvc.BuildPlan(context.Background(), PlanOptions{Root: "/docker_data"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.calls["nginx:latest"] != 1 {
		t.Fatalf("expected one digest fetch for nginx:latest, got %d", registry.calls["nginx:latest"])
	}
}
