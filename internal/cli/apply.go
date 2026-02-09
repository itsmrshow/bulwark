package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/itsmrshow/bulwark/internal/discovery"
	"github.com/itsmrshow/bulwark/internal/docker"
	"github.com/itsmrshow/bulwark/internal/executor"
	"github.com/itsmrshow/bulwark/internal/logging"
	"github.com/itsmrshow/bulwark/internal/policy"
	"github.com/itsmrshow/bulwark/internal/registry"
	"github.com/itsmrshow/bulwark/internal/state"
)

// NewApplyCommand creates the apply command
func NewApplyCommand() *cobra.Command {
	rootDefault := os.Getenv("BULWARK_ROOT")
	if rootDefault == "" {
		rootDefault = "/docker_data"
	}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply updates to managed targets",
		Long: `Applies available updates respecting policy configuration,
runs health probes, and rolls back on failure.`,
		RunE: runApply,
	}

	cmd.Flags().String("root", rootDefault, "Root directory to scan for compose projects")
	cmd.Flags().String("state", "", "Path to state database (SQLite) for persistence")
	cmd.Flags().String("db", "", "Alias for --state (SQLite path)")
	cmd.Flags().String("target", "", "Update specific target only")
	cmd.Flags().Bool("dry-run", false, "Dry-run mode (same as plan)")
	cmd.Flags().Bool("force", false, "Override policy restrictions (use with caution)")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runApply(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	stateFile, _ := cmd.Flags().GetString("state")
	dbFile, _ := cmd.Flags().GetString("db")
	targetFilter, _ := cmd.Flags().GetString("target")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	if stateFile == "" {
		stateFile = dbFile
	}

	// Initialize logger
	logger := logging.Default()

	// Show warnings
	if dryRun {
		fmt.Print("\n⚠️  DRY RUN MODE: No changes will be made\n\n")
	}

	if force {
		fmt.Print("\n⚠️  WARNING: Force mode enabled - policy restrictions will be overridden\n\n")
	}

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
	}

	// Create components
	registryClient := registry.NewClient(logger)
	policyEngine := policy.NewEngine(logger)
	discoverer := discovery.NewDiscoverer(logger, dockerClient)
	if store != nil {
		discoverer = discoverer.WithStore(store)
	}
	exec := executor.NewExecutor(dockerClient, policyEngine, store, logger, dryRun)

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

	// Check for updates and apply
	fmt.Printf("\n🔍 Checking for updates...\n\n")

	updatesApplied := 0
	updatesSkipped := 0
	updatesFailed := 0

	for _, target := range targets {
		for _, service := range target.Services {
			if !service.Labels.Enabled {
				continue
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
					Msg("Failed to fetch remote digest")
				updatesSkipped++
				continue
			}

			// Compare digests
			updateNeeded := registry.CompareDigests(service.CurrentDigest, remoteDigest)
			if !updateNeeded {
				logger.Debug().
					Str("service", service.Name).
					Msg("Service is up to date")
				continue
			}

			// Evaluate policy
			decision := policyEngine.Evaluate(ctx, &target, &service, updateNeeded)

			if !decision.Allowed && !force {
				fmt.Printf("⏭️  Skipping %s/%s: %s\n", target.Name, service.Name, decision.Reason)
				updatesSkipped++
				continue
			}

			// Apply update
			fmt.Printf("🔄 Updating %s/%s (%s)...\n", target.Name, service.Name, service.Image)

			result := exec.ExecuteUpdate(ctx, &target, &service, remoteDigest)

			if result.Success {
				fmt.Printf("✅ Updated %s/%s successfully\n", target.Name, service.Name)
				updatesApplied++
			} else if executor.IsSkipError(result.Error) {
				fmt.Printf("⏭️  Skipped %s/%s: %s\n", target.Name, service.Name, executor.SkipReason(result.Error))
				updatesSkipped++
			} else {
				fmt.Printf("❌ Failed to update %s/%s: %v\n", target.Name, service.Name, result.Error)
				updatesFailed++
			}
		}
	}

	// Summary
	fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  ✅ Updates Applied: %d\n", updatesApplied)
	fmt.Printf("  ⏭️  Updates Skipped: %d\n", updatesSkipped)
	if updatesFailed > 0 {
		fmt.Printf("  ❌ Updates Failed: %d\n", updatesFailed)
	}
	fmt.Print(strings.Repeat("=", 60) + "\n")

	if updatesFailed > 0 {
		return fmt.Errorf("%d updates failed", updatesFailed)
	}

	return nil
}
