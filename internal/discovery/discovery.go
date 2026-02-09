package discovery

import (
	"context"
	"fmt"

	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
)

// Discoverer discovers managed targets (compose projects and containers)
type Discoverer struct {
	logger           *logging.Logger
	dockerClient     *docker.Client
	composeScanner   *ComposeScanner
	containerScanner *ContainerScanner
	store            state.Store // Optional state persistence
}

// NewDiscoverer creates a new discoverer
func NewDiscoverer(logger *logging.Logger, dockerClient *docker.Client) *Discoverer {
	return &Discoverer{
		logger:           logger.WithComponent("discoverer"),
		dockerClient:     dockerClient,
		composeScanner:   NewComposeScanner(logger, dockerClient),
		containerScanner: NewContainerScanner(logger, dockerClient),
		store:            nil, // State persistence is optional
	}
}

// WithStore sets the state store for persistence
func (d *Discoverer) WithStore(store state.Store) *Discoverer {
	d.store = store
	return d
}

// Discover discovers all managed targets in the given base path
func (d *Discoverer) Discover(ctx context.Context, basePath string) ([]state.Target, error) {
	d.logger.Info().
		Str("base_path", basePath).
		Msg("Starting discovery")

	// Verify Docker connection
	if err := d.dockerClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	var allTargets []state.Target

	// Scan running containers for Bulwark labels
	// This now handles BOTH compose projects and loose containers by reading
	// labels directly from the running container (source of truth)
	containerTargets, err := d.containerScanner.ScanContainers(ctx)
	if err != nil {
		d.logger.Warn().Err(err).Msg("Failed to scan containers")
	} else {
		allTargets = append(allTargets, containerTargets...)
		d.logger.Info().
			Int("count", len(containerTargets)).
			Msg("Found targets from running containers")
	}

	// Deduplicate targets
	allTargets = d.deduplicateTargets(allTargets)

	// Persist to store if configured
	if d.store != nil {
		if err := d.persistTargets(ctx, allTargets); err != nil {
			d.logger.Warn().Err(err).Msg("Failed to persist targets to store")
		}
	}

	d.logger.Info().
		Int("total", len(allTargets)).
		Msg("Discovery complete")

	return allTargets, nil
}

// deduplicateTargets removes duplicate targets based on ID
func (d *Discoverer) deduplicateTargets(targets []state.Target) []state.Target {
	seen := make(map[string]bool)
	var unique []state.Target

	for _, target := range targets {
		if !seen[target.ID] {
			seen[target.ID] = true
			unique = append(unique, target)
		}
	}

	return unique
}

// DiscoverTarget discovers a specific target by name or ID
func (d *Discoverer) DiscoverTarget(ctx context.Context, basePath, targetID string) (*state.Target, error) {
	targets, err := d.Discover(ctx, basePath)
	if err != nil {
		return nil, err
	}

	for _, target := range targets {
		if target.ID == targetID || target.Name == targetID {
			return &target, nil
		}
	}

	return nil, fmt.Errorf("target not found: %s", targetID)
}

// CountServicesByPolicy returns statistics on services by policy
func CountServicesByPolicy(targets []state.Target) map[state.Policy]int {
	counts := make(map[state.Policy]int)

	for _, target := range targets {
		for _, service := range target.Services {
			if service.Labels.Enabled {
				counts[service.Labels.Policy]++
			}
		}
	}

	return counts
}

// CountServicesByTier returns statistics on services by tier
func CountServicesByTier(targets []state.Target) map[state.Tier]int {
	counts := make(map[state.Tier]int)

	for _, target := range targets {
		for _, service := range target.Services {
			if service.Labels.Enabled {
				counts[service.Labels.Tier]++
			}
		}
	}

	return counts
}

// GetEnabledServices returns all enabled services across all targets
func GetEnabledServices(targets []state.Target) []struct {
	Target  *state.Target
	Service *state.Service
} {
	var enabled []struct {
		Target  *state.Target
		Service *state.Service
	}

	for i := range targets {
		target := &targets[i]
		for j := range target.Services {
			service := &target.Services[j]
			if service.Labels.Enabled {
				enabled = append(enabled, struct {
					Target  *state.Target
					Service *state.Service
				}{
					Target:  target,
					Service: service,
				})
			}
		}
	}

	return enabled
}

// persistTargets saves discovered targets to the state store
func (d *Discoverer) persistTargets(ctx context.Context, targets []state.Target) error {
	for i := range targets {
		target := &targets[i]

		// Save target
		if err := d.store.SaveTarget(ctx, target); err != nil {
			d.logger.Warn().
				Err(err).
				Str("target_id", target.ID).
				Msg("Failed to save target")
			continue
		}

		// Save services
		for j := range target.Services {
			service := &target.Services[j]
			if err := d.store.SaveService(ctx, service); err != nil {
				d.logger.Warn().
					Err(err).
					Str("service_id", service.ID).
					Msg("Failed to save service")
			}
		}
	}

	d.logger.Debug().
		Int("count", len(targets)).
		Msg("Persisted targets to store")

	return nil
}
