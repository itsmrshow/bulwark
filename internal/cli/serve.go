package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewServeCommand creates the serve command
func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run as daemon with webhook server and/or scheduler",
		Long: `Runs Bulwark as a daemon process with:
- Webhook server for triggering updates via HTTP
- Scheduler for periodic update checks`,
		RunE: runServe,
	}

	cmd.Flags().Bool("no-webhook", false, "Disable webhook server")
	cmd.Flags().Bool("no-scheduler", false, "Disable scheduler")
	cmd.Flags().String("addr", ":8080", "Webhook server listen address")
	cmd.Flags().String("interval", "15m", "Scheduler interval")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	noWebhook, _ := cmd.Flags().GetBool("no-webhook")
	noScheduler, _ := cmd.Flags().GetBool("no-scheduler")
	addr, _ := cmd.Flags().GetString("addr")
	interval, _ := cmd.Flags().GetString("interval")

	fmt.Println("Starting Bulwark server...")

	if !noWebhook {
		fmt.Printf("Webhook server will listen on %s\n", addr)
	}

	if !noScheduler {
		fmt.Printf("Scheduler will run every %s\n", interval)
	}

	fmt.Println("Server not implemented yet")
	return nil
}
