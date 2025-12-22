package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start TASK_ID",
	Short: "Start a stopped agent",
	Long: `Start a previously stopped agent.

The agent must be in 'stopped' status.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("start not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
