package cli

import (
	"fmt"

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
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runDiscover(cmd *cobra.Command, args []string) error {
	root, _ := cmd.Flags().GetString("root")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	fmt.Printf("Discovering targets in %s...\n", root)

	if jsonOutput {
		fmt.Println("{\"targets\": []}")
	} else {
		fmt.Println("No targets discovered yet (not implemented)")
	}

	return nil
}
