package policy

import (
	"context"
	"fmt"

	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
)

// Engine evaluates update policies
type Engine struct {
	logger *logging.Logger
}

// NewEngine creates a new policy engine
func NewEngine(logger *logging.Logger) *Engine {
	return &Engine{
		logger: logger.WithComponent("policy"),
	}
}

// Decision represents a policy decision
type Decision struct {
	Allowed bool
	Reason  string
	Policy  state.Policy
	Tier    state.Tier
}

// Evaluate evaluates whether an update is allowed
func (e *Engine) Evaluate(ctx context.Context, target *state.Target, service *state.Service, updateAvailable bool) Decision {
	labels := service.Labels

	// Check if Bulwark is enabled
	if !labels.Enabled {
		return Decision{
			Allowed: false,
			Reason:  "Bulwark not enabled (bulwark.enabled=true required)",
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}
	}

	// Check if update is available
	if !updateAvailable {
		return Decision{
			Allowed: false,
			Reason:  "No update available",
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}
	}

	// Evaluate based on policy
	switch labels.Policy {
	case state.PolicyNotify:
		return Decision{
			Allowed: false,
			Reason:  "Policy is 'notify' - manual updates only",
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}

	case state.PolicySafe:
		// Safe policy: check tier and probe configuration
		if labels.Tier == state.TierStateful {
			return Decision{
				Allowed: false,
				Reason:  "Safe policy blocks stateful service updates (use aggressive to override)",
				Policy:  labels.Policy,
				Tier:    labels.Tier,
			}
		}

		// Check if probes are configured for safe policy
		if labels.Probe.Type == state.ProbeTypeNone {
			e.logger.Warn().
				Str("service", service.Name).
				Msg("Safe policy without probes - update allowed but risky")
		}

		return Decision{
			Allowed: true,
			Reason:  "Safe policy allows stateless updates with probes",
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}

	case state.PolicyAggressive:
		// Aggressive policy: always allow (with warning for stateful)
		if labels.Tier == state.TierStateful {
			e.logger.Warn().
				Str("service", service.Name).
				Str("image", service.Image).
				Msg("Aggressive policy on stateful service - high risk!")
		}

		return Decision{
			Allowed: true,
			Reason:  "Aggressive policy allows all updates",
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}

	default:
		return Decision{
			Allowed: false,
			Reason:  fmt.Sprintf("Unknown policy: %s", labels.Policy),
			Policy:  labels.Policy,
			Tier:    labels.Tier,
		}
	}
}

// EvaluateAll evaluates policy for all services
func (e *Engine) EvaluateAll(ctx context.Context, checks []state.UpdateCheck) []state.UpdateCheck {
	for i := range checks {
		check := &checks[i]
		decision := e.Evaluate(ctx, check.Target, check.Service, check.UpdateNeeded)
		check.PolicyAllows = decision.Allowed
		check.Reason = decision.Reason
	}
	return checks
}

// ShouldRollback determines if a failed update should be rolled back
func (e *Engine) ShouldRollback(ctx context.Context, result *state.UpdateResult) bool {
	// Always rollback on failure for safe and aggressive policies
	if !result.Success {
		return true
	}

	// If probes failed, rollback
	for _, probe := range result.ProbeResults {
		if !probe.Success {
			return true
		}
	}

	return false
}

// ValidateProbeConfiguration checks if probe configuration is valid
func (e *Engine) ValidateProbeConfiguration(labels state.Labels) []string {
	var warnings []string

	if !labels.Enabled {
		return warnings
	}

	// Check probe configuration based on type
	switch labels.Probe.Type {
	case state.ProbeTypeHTTP:
		if labels.Probe.HTTPUrl == "" {
			warnings = append(warnings, "HTTP probe configured but no URL provided")
		}
		if labels.Probe.HTTPStatus == 0 {
			warnings = append(warnings, "HTTP probe status code not set, defaulting to 200")
		}

	case state.ProbeTypeTCP:
		if labels.Probe.TCPHost == "" {
			warnings = append(warnings, "TCP probe configured but no host provided")
		}
		if labels.Probe.TCPPort == 0 {
			warnings = append(warnings, "TCP probe configured but no port provided")
		}

	case state.ProbeTypeLog:
		if labels.Probe.LogPattern == "" {
			warnings = append(warnings, "Log probe configured but no pattern provided")
		}

	case state.ProbeTypeStability:
		if labels.Probe.StabilitySec == 0 {
			warnings = append(warnings, "Stability probe configured but no duration provided")
		}

	case state.ProbeTypeNone:
		if labels.Policy == state.PolicySafe {
			warnings = append(warnings, "Safe policy without probes is risky")
		}
	}

	// Check policy and tier combination
	if labels.Policy == state.PolicyAggressive && labels.Tier == state.TierStateful {
		warnings = append(warnings, "Aggressive policy on stateful service is very risky")
	}

	return warnings
}
