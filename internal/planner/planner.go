package planner

import (
	"context"
	"fmt"
	"time"

	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/registry"
	"github.com/itsmrshow/bulwark/internal/state"
)

const (
	RiskSafe         = "safe"
	RiskNotifyOnly   = "notify"
	RiskStateful     = "stateful"
	RiskProbeMissing = "probe_missing"
)

// PlanOptions configures plan generation.
type PlanOptions struct {
	Root            string
	TargetFilter    string
	IncludeDisabled bool
}

// Plan represents a structured update plan.
type Plan struct {
	GeneratedAt  time.Time  `json:"generated_at"`
	TargetCount  int        `json:"target_count"`
	ServiceCount int        `json:"service_count"`
	UpdateCount  int        `json:"update_count"`
	AllowedCount int        `json:"allowed_count"`
	Items        []PlanItem `json:"items"`
}

// PlanItem represents one service decision.
type PlanItem struct {
	TargetID        string            `json:"target_id"`
	TargetName      string            `json:"target_name"`
	TargetType      state.TargetType  `json:"target_type"`
	ServiceID       string            `json:"service_id"`
	ServiceName     string            `json:"service_name"`
	Image           string            `json:"image"`
	CurrentDigest   string            `json:"current_digest"`
	RemoteDigest    string            `json:"remote_digest"`
	UpdateAvailable bool              `json:"update_available"`
	Allowed         bool              `json:"allowed"`
	Policy          state.Policy      `json:"policy"`
	Tier            state.Tier        `json:"tier"`
	Probe           state.ProbeConfig `json:"probe"`
	Reason          string            `json:"reason"`
	Risk            string            `json:"risk"`
	Warnings        []string          `json:"warnings,omitempty"`
	Target          *state.Target     `json:"-"`
	Service         *state.Service    `json:"-"`
}

// Planner builds structured plans.
type Planner struct {
	logger       *logging.Logger
	discoverer   discoverer
	registry     digestFetcher
	policyEngine *policy.Engine
}

type discoverer interface {
	Discover(ctx context.Context, basePath string) ([]state.Target, error)
	DiscoverTarget(ctx context.Context, basePath, targetID string) (*state.Target, error)
}

type digestFetcher interface {
	FetchDigest(ctx context.Context, image string) (string, error)
}

// NewPlanner creates a planner.
func NewPlanner(logger *logging.Logger, discoverer discoverer, registry digestFetcher, policyEngine *policy.Engine) *Planner {
	if logger == nil {
		logger = logging.Default()
	}
	return &Planner{
		logger:       logger.WithComponent("planner"),
		discoverer:   discoverer,
		registry:     registry,
		policyEngine: policyEngine,
	}
}

// BuildPlan creates a plan for targets.
func (p *Planner) BuildPlan(ctx context.Context, opts PlanOptions) (*Plan, error) {
	var targets []state.Target
	var err error

	if opts.TargetFilter != "" {
		var target *state.Target
		target, err = p.discoverer.DiscoverTarget(ctx, opts.Root, opts.TargetFilter)
		if err != nil {
			return nil, err
		}
		targets = []state.Target{*target}
	} else {
		targets, err = p.discoverer.Discover(ctx, opts.Root)
		if err != nil {
			return nil, err
		}
	}

	plan := &Plan{
		GeneratedAt: time.Now().UTC(),
		TargetCount: len(targets),
		Items:       []PlanItem{},
	}

	for i := range targets {
		target := &targets[i]
		for j := range target.Services {
			service := &target.Services[j]
			if !service.Labels.Enabled && !opts.IncludeDisabled {
				continue
			}
			plan.ServiceCount++

			item := PlanItem{
				TargetID:      target.ID,
				TargetName:    target.Name,
				TargetType:    target.Type,
				ServiceID:     service.ID,
				ServiceName:   service.Name,
				Image:         service.Image,
				CurrentDigest: service.CurrentDigest,
				Policy:        service.Labels.Policy,
				Tier:          service.Labels.Tier,
				Probe:         service.Labels.Probe,
				Target:        target,
				Service:       service,
			}

			item.Risk = riskFromLabels(service.Labels)

			remoteDigest, digestErr := p.registry.FetchDigest(ctx, service.Image)
			if digestErr != nil {
				item.UpdateAvailable = false
				item.Allowed = false
				item.Reason = fmt.Sprintf("Failed to fetch digest: %v", digestErr)
				item.Warnings = p.policyEngine.ValidateProbeConfiguration(service.Labels)
				plan.Items = append(plan.Items, item)
				continue
			}

			item.RemoteDigest = remoteDigest

			updateAvailable := false
			reason := ""
			if service.CurrentDigest == "" {
				updateAvailable = true
				reason = "No current digest (container not running)"
			} else if registry.CompareDigests(service.CurrentDigest, remoteDigest) {
				updateAvailable = true
				reason = "Digest mismatch - update available"
			} else {
				updateAvailable = false
				reason = "Digests match - up to date"
			}

			decision := p.policyEngine.Evaluate(ctx, target, service, updateAvailable)
			item.UpdateAvailable = updateAvailable
			item.Allowed = decision.Allowed
			if updateAvailable {
				item.Reason = decision.Reason
			} else {
				item.Reason = reason
			}
			item.Warnings = p.policyEngine.ValidateProbeConfiguration(service.Labels)

			if item.UpdateAvailable {
				plan.UpdateCount++
				if item.Allowed {
					plan.AllowedCount++
				}
			}

			plan.Items = append(plan.Items, item)
		}
	}

	return plan, nil
}

func riskFromLabels(labels state.Labels) string {
	if labels.Policy == state.PolicyNotify {
		return RiskNotifyOnly
	}
	if labels.Tier == state.TierStateful {
		return RiskStateful
	}
	if labels.Probe.Type == state.ProbeTypeNone {
		return RiskProbeMissing
	}
	return RiskSafe
}

// HistoryFilter filters history entries.
type HistoryFilter struct {
	TargetID  string
	ServiceID string
	Result    string
}

// HistoryItem represents a record for the history endpoint.
type HistoryItem struct {
	TargetID     string    `json:"target_id"`
	ServiceID    string    `json:"service_id"`
	ServiceName  string    `json:"service_name"`
	OldDigest    string    `json:"old_digest"`
	NewDigest    string    `json:"new_digest"`
	Success      bool      `json:"success"`
	RolledBack   bool      `json:"rolled_back"`
	ErrorMessage string    `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	ProbesPassed int       `json:"probes_passed"`
	ProbesFailed int       `json:"probes_failed"`
	DurationSec  float64   `json:"duration_sec"`
}

// MapHistory converts update results to history items.
func MapHistory(results []state.UpdateResult) []HistoryItem {
	items := make([]HistoryItem, 0, len(results))
	for _, result := range results {
		message := ""
		if result.Error != nil {
			message = result.Error.Error()
		}

		completedAt := result.CompletedAt
		durationSec := 0.0
		if completedAt.IsZero() || completedAt.Before(result.StartedAt) {
			completedAt = result.StartedAt
		} else {
			durationSec = completedAt.Sub(result.StartedAt).Seconds()
		}

		probesPassed := 0
		probesFailed := 0
		for _, probe := range result.ProbeResults {
			if probe.Success {
				probesPassed++
			} else {
				probesFailed++
			}
		}
		items = append(items, HistoryItem{
			TargetID:     result.TargetID,
			ServiceID:    result.ServiceID,
			ServiceName:  result.ServiceName,
			OldDigest:    result.OldDigest,
			NewDigest:    result.NewDigest,
			Success:      result.Success,
			RolledBack:   result.RollbackPerformed,
			ErrorMessage: message,
			StartedAt:    result.StartedAt,
			CompletedAt:  completedAt,
			ProbesPassed: probesPassed,
			ProbesFailed: probesFailed,
			DurationSec:  durationSec,
		})
	}
	return items
}

// FilterHistory filters history items based on filters.
func FilterHistory(items []HistoryItem, filter HistoryFilter) []HistoryItem {
	result := make([]HistoryItem, 0, len(items))
	for _, item := range items {
		if filter.TargetID != "" && item.TargetID != filter.TargetID {
			continue
		}
		if filter.ServiceID != "" && item.ServiceID != filter.ServiceID {
			continue
		}
		if filter.Result != "" {
			switch filter.Result {
			case "success":
				if !item.Success {
					continue
				}
			case "failed":
				if item.Success {
					continue
				}
			case "rolled_back":
				if !item.RolledBack {
					continue
				}
			}
		}
		result = append(result, item)
	}
	return result
}
