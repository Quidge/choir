package cmd

import (
	"fmt"
	"os"

	"github.com/Quidge/choir/cmd/env"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Global flags
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "choir",
	Short: "Manage isolated environments for development work",
	Long: `Choir manages isolated development environments for running
AI coding assistants in parallel. Each environment operates in its own
workspace with full isolation, enabling multiple concurrent workstreams
on the same codebase without conflicts.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.AddCommand(env.Cmd)
}
