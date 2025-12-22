package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach TASK_ID",
	Short: "Connect to a running agent",
	Long: `Connect to an existing agent's shell.

If the agent is stopped, it will be started first.
When you exit the shell, the VM continues running.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("attach not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
}
