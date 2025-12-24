// Package env provides the `choir env` command group for managing environments.
package env

import (
	"github.com/spf13/cobra"
)

// Cmd is the parent command for environment management.
var Cmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environments",
	Long: `Manage isolated environments for development work.

An environment is an isolated workspace (worktree, VM, etc.) where work happens.
Environments can be created, attached to, listed, and removed.`,
}

func init() {
	Cmd.AddCommand(createCmd)
	Cmd.AddCommand(attachCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(rmCmd)
	Cmd.AddCommand(statusCmd)
}
