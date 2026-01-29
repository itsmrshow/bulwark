package planner

import (
	"context"
	"testing"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/policy"
	"github.com/yourusername/bulwark/internal/state"
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
