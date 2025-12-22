package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm TASK_ID",
	Short: "Remove an agent",
	Long: `Remove an agent and destroy its VM.

If the agent is running, you will be prompted for confirmation
unless --force is specified.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("rm not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("force", "f", false, "remove without confirmation")
}
