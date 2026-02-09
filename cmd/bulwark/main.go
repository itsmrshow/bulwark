package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/itsmrshow/bulwark/internal/cli"
	"github.com/itsmrshow/bulwark/internal/logging"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	// Initialize default logger
	logging.Init(logging.Config{
		Level:         getEnv("BULWARK_LOG_LEVEL", "info"),
		Format:        getEnv("BULWARK_LOG_FORMAT", "console"),
		RedactSecrets: true,
	})

	rootCmd := &cobra.Command{
		Use:   "bulwark",
		Short: "Bulwark - Safe, policy-driven Docker container updater",
		Long: `Bulwark is a safe, policy-driven Docker container updater that detects
image updates by digest and applies guarded updates with rollback capability.

It discovers Docker Compose projects and labeled containers, checks for updates
using digest comparison, and applies updates safely with health probes.`,
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
	}

	// Global flags
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "console", "Log format (console, json)")
	rootCmd.PersistentFlags().String("config", "", "Config file path")

	// Add commands
	rootCmd.AddCommand(cli.NewDiscoverCommand())
	rootCmd.AddCommand(cli.NewCheckCommand())
	rootCmd.AddCommand(cli.NewPlanCommand())
	rootCmd.AddCommand(cli.NewApplyCommand())
	rootCmd.AddCommand(cli.NewServeCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
