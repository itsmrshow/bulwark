package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
	"gopkg.in/yaml.v3"
)

// ComposeScanner scans for Docker Compose projects
type ComposeScanner struct {
	logger        *logging.Logger
	dockerClient  *docker.Client
	composeRunner *docker.ComposeRunner
}

// NewComposeScanner creates a new compose scanner
func NewComposeScanner(logger *logging.Logger, dockerClient *docker.Client) *ComposeScanner {
	return &ComposeScanner{
		logger:        logger.WithComponent("compose-scanner"),
		dockerClient:  dockerClient,
		composeRunner: docker.NewComposeRunner(),
	}
}

// ComposeFile represents a parsed docker-compose.yml file
type ComposeFile struct {
	Version  string                       `yaml:"version"`
	Services map[string]ComposeService    `yaml:"services"`
}

// ComposeService represents a service in docker-compose.yml
type ComposeService struct {
	Image       string             `yaml:"image"`
	Labels      interface{}        `yaml:"labels"` // Can be map or array
	HealthCheck *HealthCheckConfig `yaml:"healthcheck,omitempty"`
}

// HealthCheckConfig represents Docker healthcheck configuration
type HealthCheckConfig struct {
	Test        interface{} `yaml:"test"`
	Interval    string      `yaml:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty"`
}

// ScanProjects scans for Docker Compose projects in the given base path
func (s *ComposeScanner) ScanProjects(ctx context.Context, basePath string) ([]state.Target, error) {
	s.logger.Info().Str("base_path", basePath).Msg("Scanning for compose projects")

	// Find all compose files
	composeFiles, err := docker.FindComposeFiles(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find compose files: %w", err)
	}

	s.logger.Info().Int("count", len(composeFiles)).Msg("Found compose files")

	var targets []state.Target

	for _, composePath := range composeFiles {
		target, err := s.parseComposeFile(ctx, composePath)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("path", composePath).
				Msg("Failed to parse compose file")
			continue
		}

		// Only include if at least one service is enabled
		hasEnabled := false
		for _, service := range target.Services {
			if service.Labels.Enabled {
				hasEnabled = true
				break
			}
		}

		if hasEnabled {
			targets = append(targets, *target)
			s.logger.Info().
				Str("target", target.Name).
				Int("services", len(target.Services)).
				Msg("Found enabled compose project")
		}
	}

	return targets, nil
}

// parseComposeFile parses a single docker-compose.yml file
func (s *ComposeScanner) parseComposeFile(ctx context.Context, composePath string) (*state.Target, error) {
	// Read compose file
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var composeFile ComposeFile
	if err := yaml.Unmarshal(data, &composeFile); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Create target
	projectName := filepath.Base(filepath.Dir(composePath))
	target := &state.Target{
		ID:        state.GenerateTargetID(state.TargetTypeCompose, projectName, composePath),
		Type:      state.TargetTypeCompose,
		Name:      projectName,
		Path:      composePath,
		Services:  []state.Service{},
		Labels:    state.DefaultLabels(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Parse services
	for serviceName, composeService := range composeFile.Services {
		if composeService.Image == "" {
			// Skip services without an image (build-only services)
			continue
		}

		// Convert labels to map[string]string (handles both map and array formats)
		labelMap := convertLabelsToMap(composeService.Labels)

		// Parse labels
		labels := ParseLabels(labelMap, composeService.Image)

		// Get current digest from Docker if container is running
		digest := s.getCurrentDigest(ctx, target.Name, serviceName)

		// Parse healthcheck
		var healthCheck *state.HealthCheck
		if composeService.HealthCheck != nil {
			healthCheck = parseHealthCheck(composeService.HealthCheck)
		}

		service := state.Service{
			ID:            state.GenerateServiceID(target.ID, serviceName),
			TargetID:      target.ID,
			Name:          serviceName,
			Image:         composeService.Image,
			CurrentDigest: digest,
			Labels:        labels,
			HealthCheck:   healthCheck,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		target.Services = append(target.Services, service)
	}

	return target, nil
}

// getCurrentDigest gets the current digest of a running container
func (s *ComposeScanner) getCurrentDigest(ctx context.Context, projectName, serviceName string) string {
	// List containers with label filters
	containers, err := s.dockerClient.ListContainers(ctx, false)
	if err != nil {
		return ""
	}

	// Find container for this service
	for _, container := range containers {
		// Check if it's part of the compose project
		if container.Labels["com.docker.compose.project"] == projectName &&
			container.Labels["com.docker.compose.service"] == serviceName {

			// Inspect to get image digest
			inspect, err := s.dockerClient.InspectContainer(ctx, container.ID)
			if err != nil {
				continue
			}

			// Return the image ID (digest)
			return inspect.Image
		}
	}

	return ""
}

// convertLabelsToMap converts labels from interface{} (map or array) to map[string]string
func convertLabelsToMap(labels interface{}) map[string]string {
	result := make(map[string]string)

	if labels == nil {
		return result
	}

	switch v := labels.(type) {
	case map[string]interface{}:
		// Map format: labels: {key: value}
		for key, val := range v {
			if str, ok := val.(string); ok {
				result[key] = str
			}
		}
	case []interface{}:
		// Array format: labels: ["key=value", ...]
		for _, item := range v {
			if str, ok := item.(string); ok {
				// Split on first '=' to handle values with '=' in them
				parts := strings.SplitN(str, "=", 2)
				if len(parts) == 2 {
					result[parts[0]] = parts[1]
				}
			}
		}
	case map[interface{}]interface{}:
		// YAML can also parse as map[interface{}]interface{}
		for key, val := range v {
			if keyStr, ok := key.(string); ok {
				if valStr, ok := val.(string); ok {
					result[keyStr] = valStr
				}
			}
		}
	}

	return result
}

// parseHealthCheck converts compose healthcheck to our format
func parseHealthCheck(hc *HealthCheckConfig) *state.HealthCheck {
	if hc == nil {
		return nil
	}

	healthCheck := &state.HealthCheck{
		Retries: hc.Retries,
	}

	// Parse test command
	switch test := hc.Test.(type) {
	case []interface{}:
		for _, t := range test {
			if str, ok := t.(string); ok {
				healthCheck.Test = append(healthCheck.Test, str)
			}
		}
	case string:
		healthCheck.Test = []string{"CMD-SHELL", test}
	}

	// Parse durations
	if hc.Interval != "" {
		if d, err := time.ParseDuration(hc.Interval); err == nil {
			healthCheck.Interval = d
		}
	}
	if hc.Timeout != "" {
		if d, err := time.ParseDuration(hc.Timeout); err == nil {
			healthCheck.Timeout = d
		}
	}
	if hc.StartPeriod != "" {
		if d, err := time.ParseDuration(hc.StartPeriod); err == nil {
			healthCheck.StartPeriod = d
		}
	}

	return healthCheck
}

