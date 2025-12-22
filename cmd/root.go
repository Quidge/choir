package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Global flags
	backend string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "choir",
	Short: "Manage isolated VM environments for AI agents",
	Long: `Choir manages isolated development environments ("agents") for running
AI coding assistants in parallel. Each agent operates in its own VM with
full filesystem and network isolation, enabling multiple concurrent
workstreams on the same codebase without conflicts.`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&backend, "backend", "", "override default backend")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}
