package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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

	cmd.Flags().String("target", "", "Check specific target only")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")

	if target != "" {
		fmt.Printf("Checking target: %s\n", target)
	} else {
		fmt.Println("Checking all targets...")
	}

	fmt.Println("No updates available (not implemented)")
	return nil
}
