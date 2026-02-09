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
	"github.com/itsmrshow/bulwark/internal/planner"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/registry"
	"github.com/itsmrshow/bulwark/internal/state"
)

// NewPlanCommand creates the plan command
func NewPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show what would be updated (dry-run)",
		Long: `Shows what services would be updated if apply were run,
including policy decisions and probe configurations.`,
		RunE: runPlan,
	}

	cmd.Flags().String("root", "/docker_data", "Root directory to scan for compose projects")
	cmd.Flags().String("state", "", "Path to state database (SQLite) for persistence")
	cmd.Flags().String("target", "", "Plan for specific target only")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runPlan(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	stateFile, _ := cmd.Flags().GetString("state")
	target, _ := cmd.Flags().GetString("target")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	logger := logging.Default()

	dockerClient, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = dockerClient.Close() }()

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
	}

	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if store != nil {
		discoverer = discoverer.WithStore(store)
	}

	registryClient := registry.NewClient(logger)
	policyEngine := policy.NewEngine(logger)
	plannerSvc := planner.NewPlanner(logger, discoverer, registryClient, policyEngine)

	plan, err := plannerSvc.BuildPlan(context.Background(), planner.PlanOptions{
		Root:            root,
		TargetFilter:    target,
		IncludeDisabled: false,
	})
	if err != nil {
		return fmt.Errorf("plan failed: %w", err)
	}

	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(plan)
	}

	if target != "" {
		fmt.Printf("Planning updates for target: %s\n\n", target)
	} else {
		fmt.Println("Planning updates for all targets...")
		fmt.Println()
	}

	fmt.Printf("%-20s %-20s %-10s %-10s %-12s %-8s %s\n",
		"TARGET", "SERVICE", "POLICY", "TIER", "UPDATE?", "ALLOWED", "REASON")
	fmt.Println(strings.Repeat("-", 110))

	for _, item := range plan.Items {
		updateStatus := "No"
		if item.UpdateAvailable {
			updateStatus = "Yes"
		}
		allowedStatus := "No"
		if item.Allowed {
			allowedStatus = "Yes"
		}
		targetName := item.TargetName
		if len(targetName) > 20 {
			targetName = targetName[:17] + "..."
		}
		serviceName := item.ServiceName
		if len(serviceName) > 20 {
			serviceName = serviceName[:17] + "..."
		}
		reason := item.Reason
		if len(reason) > 50 {
			reason = reason[:47] + "..."
		}
		fmt.Printf("%-20s %-20s %-10s %-10s %-12s %-8s %s\n",
			targetName,
			serviceName,
			string(item.Policy),
			string(item.Tier),
			updateStatus,
			allowedStatus,
			reason,
		)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Targets: %d\n", plan.TargetCount)
	fmt.Printf("  Services: %d\n", plan.ServiceCount)
	fmt.Printf("  Updates Available: %d\n", plan.UpdateCount)
	fmt.Printf("  Updates Allowed: %d\n", plan.AllowedCount)

	if plan.UpdateCount > 0 {
		return fmt.Errorf("updates available")
	}

	return nil
}
