package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status TASK_ID",
	Short: "Show detailed agent status",
	Long: `Show detailed status information for an agent.

Displays task ID, backend, status, branch, base branch, repository,
remote URL, creation time, and resource allocation.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("status not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
