package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs TASK_ID",
	Short: "Show agent provisioning logs",
	Long: `Show provisioning and setup logs for an agent.

Useful for debugging failed spawns or reviewing setup command output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("logs not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "stream logs (if agent is provisioning)")
}
