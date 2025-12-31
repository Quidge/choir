package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop TASK_ID",
	Short: "Stop a running agent",
	Long: `Stop a running agent without removing it.

The agent can be restarted later with 'choir start'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("stop not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
