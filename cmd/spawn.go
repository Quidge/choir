package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn TASK_ID",
	Short: "Create and start a new agent",
	Long: `Create and start a new agent with the given task ID.

The agent runs in an isolated VM with a clone of the current repository
on a dedicated branch (agent/<task-id> by default).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		return fmt.Errorf("spawn not implemented: %s", taskID)
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)

	spawnCmd.Flags().String("base", "", "base branch to spawn from")
	spawnCmd.Flags().String("prompt", "", "initial prompt to display when agent starts")
	spawnCmd.Flags().String("task-file", "", "file containing task description")
	spawnCmd.Flags().Bool("rm", false, "remove agent automatically when shell exits")
	spawnCmd.Flags().Int("cpus", 0, "override CPU allocation")
	spawnCmd.Flags().String("memory", "", "override memory allocation")
	spawnCmd.Flags().Bool("no-setup", false, "skip setup commands from project config")
}
