package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/itsmrshow/bulwark/internal/discovery"
	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/state"
	"github.com/spf13/cobra"
)

// NewDiscoverCommand creates the discover command
func NewDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover managed targets (compose projects and labeled containers)",
		Long: `Discovers all Docker Compose projects under the configured root directory
and all containers with bulwark.enabled=true labels.`,
		RunE: runDiscover,
	}

	cmd.Flags().String("root", "/docker_data", "Root directory to scan for compose projects")
	cmd.Flags().String("state", "", "Path to state database (SQLite) for persistence")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().Bool("show-disabled", false, "Show services with bulwark.enabled=false")

	return cmd
}

func runDiscover(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	stateFile, _ := cmd.Flags().GetString("state")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showDisabled, _ := cmd.Flags().GetBool("show-disabled")

	// Initialize logger
	logger := logging.Default()

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = dockerClient.Close() }()

	// Create state store if path provided
	var store state.Store
	if stateFile != "" {
		sqliteStore, err := state.NewSQLiteStore(stateFile, logger)
		if err != nil {
			return fmt.Errorf("failed to create state store: %w", err)
		}
		defer func() { _ = sqliteStore.Close() }()

		ctx := context.Background()
		if err := sqliteStore.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize state store: %w", err)
		}

		store = sqliteStore
		logger.Info().Str("path", stateFile).Msg("State persistence enabled")
	}

	// Create discoverer
	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if store != nil {
		discoverer = discoverer.WithStore(store)
	}

	// Run discovery
	ctx := context.Background()
	targets, err := discoverer.Discover(ctx, root)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Output results
	if jsonOutput {
		return outputDiscoveryJSON(targets)
	}

	return outputDiscoveryTable(targets, showDisabled)
}

func outputDiscoveryJSON(targets []state.Target) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"targets": targets,
	})
}

func outputDiscoveryTable(targets []state.Target, showDisabled bool) error {
	if len(targets) == 0 {
		fmt.Println("No targets discovered.")
		fmt.Println("\nTo enable Bulwark management, add labels to your services:")
		fmt.Println("  bulwark.enabled=true")
		fmt.Println("  bulwark.policy=safe")
		return nil
	}

	fmt.Printf("\nDiscovered Targets:\n\n")
	fmt.Printf("%-10s %-20s %-20s %-30s %-8s %-10s %-10s %-10s\n",
		"TYPE", "TARGET", "SERVICE", "IMAGE", "ENABLED", "POLICY", "TIER", "PROBE")
	fmt.Println(strings.Repeat("-", 120))

	enabledCount := 0
	disabledCount := 0

	for _, target := range targets {
		for _, service := range target.Services {
			if !service.Labels.Enabled && !showDisabled {
				disabledCount++
				continue
			}

			enabled := "No"
			if service.Labels.Enabled {
				enabled = "Yes"
				enabledCount++
			} else {
				disabledCount++
			}

			// Truncate image if too long
			image := service.Image
			if len(image) > 30 {
				image = image[:27] + "..."
			}

			// Get probe type
			probeType := string(service.Labels.Probe.Type)
			if probeType == "" || probeType == "none" {
				probeType = "-"
			}

			// Truncate target/service if too long
			targetName := target.Name
			if len(targetName) > 20 {
				targetName = targetName[:17] + "..."
			}
			serviceName := service.Name
			if len(serviceName) > 20 {
				serviceName = serviceName[:17] + "..."
			}

			fmt.Printf("%-10s %-20s %-20s %-30s %-8s %-10s %-10s %-10s\n",
				string(target.Type),
				targetName,
				serviceName,
				image,
				enabled,
				string(service.Labels.Policy),
				string(service.Labels.Tier),
				probeType)
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Targets: %d\n", len(targets))
	fmt.Printf("  Enabled Services: %d\n", enabledCount)
	if disabledCount > 0 {
		fmt.Printf("  Disabled Services: %d", disabledCount)
		if !showDisabled {
			fmt.Printf(" (use --show-disabled to see)")
		}
		fmt.Println()
	}

	return nil
}
