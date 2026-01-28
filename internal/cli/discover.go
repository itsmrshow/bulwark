package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/yourusername/bulwark/internal/discovery"
	"github.com/yourusername/bulwark/internal/docker"
	"github.com/yourusername/bulwark/internal/logging"
	"github.com/yourusername/bulwark/internal/state"
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
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().Bool("show-disabled", false, "Show services with bulwark.enabled=false")

	return cmd
}

func runDiscover(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showDisabled, _ := cmd.Flags().GetBool("show-disabled")

	// Initialize logger
	logger := logging.Default()

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	// Create discoverer
	discoverer := discovery.NewDiscoverer(logger, dockerClient)

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

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Target", "Service", "Image", "Enabled", "Policy", "Tier", "Probe"})
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

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
			if len(image) > 40 {
				image = image[:37] + "..."
			}

			// Get probe type
			probeType := string(service.Labels.Probe.Type)
			if probeType == "" || probeType == "none" {
				probeType = "-"
			}

			table.Append([]string{
				string(target.Type),
				target.Name,
				service.Name,
				image,
				enabled,
				string(service.Labels.Policy),
				string(service.Labels.Tier),
				probeType,
			})
		}
	}

	fmt.Printf("\nDiscovered Targets:\n\n")
	table.Render()

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
