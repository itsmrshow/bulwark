package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/itsmrshow/bulwark/internal/discovery"
	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/registry"
	"github.com/itsmrshow/bulwark/internal/state"
)

// NewCheckCommand creates the check command
func NewCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check for available updates",
		Long: `Checks for available image updates by comparing local digests
with remote registry digests. Respects policy configuration.`,
		RunE: runCheck,
	}

	cmd.Flags().String("root", "/docker_data", "Root directory to scan for compose projects")
	cmd.Flags().String("target", "", "Check specific target only")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.Flags().Bool("show-all", false, "Show all services, even without updates")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	targetFilter, _ := cmd.Flags().GetString("target")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("show-all")

	// Initialize logger
	logger := logging.Default()

	// Create clients
	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = dockerClient.Close() }()

	registryClient := registry.NewClient(logger)
	policyEngine := policy.NewEngine(logger)
	discoverer := discovery.NewDiscoverer(logger, dockerClient)

	// Run discovery
	ctx := context.Background()
	var targets []state.Target

	if targetFilter != "" {
		target, err := discoverer.DiscoverTarget(ctx, root, targetFilter)
		if err != nil {
			return fmt.Errorf("failed to discover target: %w", err)
		}
		targets = []state.Target{*target}
	} else {
		targets, err = discoverer.Discover(ctx, root)
		if err != nil {
			return fmt.Errorf("discovery failed: %w", err)
		}
	}

	if len(targets) == 0 {
		fmt.Println("No targets discovered. Enable Bulwark on your services with bulwark.enabled=true")
		return nil
	}

	// Check each service for updates
	var checks []state.UpdateCheck

	for _, target := range targets {
		for _, service := range target.Services {
			if !service.Labels.Enabled {
				continue
			}

			check := state.UpdateCheck{
				Target:       &target,
				Service:      &service,
				UpdateNeeded: false,
				PolicyAllows: false,
			}

			// Fetch remote digest
			logger.Debug().
				Str("service", service.Name).
				Str("image", service.Image).
				Msg("Fetching remote digest")

			remoteDigest, err := registryClient.FetchDigest(ctx, service.Image)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("service", service.Name).
					Str("image", service.Image).
					Msg("Failed to fetch remote digest")
				check.Reason = fmt.Sprintf("Failed to fetch digest: %v", err)
				checks = append(checks, check)
				continue
			}

			check.RemoteDigest = remoteDigest

			// Compare digests
			if service.CurrentDigest == "" {
				check.UpdateNeeded = true
				check.Reason = "No current digest (container not running)"
			} else if registry.CompareDigests(service.CurrentDigest, remoteDigest) {
				check.UpdateNeeded = true
				check.Reason = "Digest mismatch - update available"
			} else {
				check.UpdateNeeded = false
				check.Reason = "Digests match - up to date"
			}

			// Evaluate policy
			decision := policyEngine.Evaluate(ctx, &target, &service, check.UpdateNeeded)
			check.PolicyAllows = decision.Allowed
			if check.UpdateNeeded {
				check.Reason = decision.Reason
			}

			checks = append(checks, check)
		}
	}

	// Output results
	if jsonOutput {
		return outputCheckJSON(checks)
	}

	return outputCheckTable(checks, showAll)
}

func outputCheckJSON(checks []state.UpdateCheck) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"checks": checks,
	})
}

func outputCheckTable(checks []state.UpdateCheck, showAll bool) error {
	fmt.Printf("\nUpdate Check Results:\n\n")
	fmt.Printf("%-20s %-20s %-15s %-12s %-12s %s\n",
		"TARGET", "SERVICE", "POLICY", "UPDATE?", "ALLOWED?", "REASON")
	fmt.Println(strings.Repeat("-", 120))

	updatesAvailable := 0
	updatesAllowed := 0
	upToDate := 0

	for _, check := range checks {
		// Skip up-to-date services unless showAll
		if !check.UpdateNeeded && !showAll {
			upToDate++
			continue
		}

		if check.UpdateNeeded {
			updatesAvailable++
			if check.PolicyAllows {
				updatesAllowed++
			}
		} else {
			upToDate++
		}

		// Truncate names if too long
		targetName := check.Target.Name
		if len(targetName) > 20 {
			targetName = targetName[:17] + "..."
		}
		serviceName := check.Service.Name
		if len(serviceName) > 20 {
			serviceName = serviceName[:17] + "..."
		}

		updateStatus := "No"
		if check.UpdateNeeded {
			updateStatus = "Yes"
		}

		allowedStatus := "No"
		if check.PolicyAllows {
			allowedStatus = "Yes"
		}

		// Truncate reason
		reason := check.Reason
		if len(reason) > 50 {
			reason = reason[:47] + "..."
		}

		fmt.Printf("%-20s %-20s %-15s %-12s %-12s %s\n",
			targetName,
			serviceName,
			string(check.Service.Labels.Policy),
			updateStatus,
			allowedStatus,
			reason)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Services Checked: %d\n", len(checks))
	fmt.Printf("  Updates Available: %d\n", updatesAvailable)
	fmt.Printf("  Updates Allowed by Policy: %d\n", updatesAllowed)
	fmt.Printf("  Up to Date: %d", upToDate)
	if !showAll && upToDate > 0 {
		fmt.Printf(" (use --show-all to see)")
	}
	fmt.Println()

	// Return non-zero exit code if updates are available
	if updatesAvailable > 0 {
		return fmt.Errorf("updates available")
	}

	return nil
}
