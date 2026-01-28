package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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

	cmd.Flags().String("target", "", "Plan for specific target only")
	cmd.Flags().Bool("json", false, "Output as JSON")

	return cmd
}

func runPlan(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")

	if target != "" {
		fmt.Printf("Planning updates for target: %s\n", target)
	} else {
		fmt.Println("Planning updates for all targets...")
	}

	fmt.Println("Nothing to update (not implemented)")
	return nil
}
