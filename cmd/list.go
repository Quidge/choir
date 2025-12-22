package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	Long: `List all agents, optionally filtered by backend or repository.

By default, removed and failed agents are hidden. Use --all to show them.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("list not implemented")
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().String("backend", "", "filter by backend")
	listCmd.Flags().Bool("repo", false, "filter by current repository")
	listCmd.Flags().Bool("all", false, "include removed/failed agents")
	listCmd.Flags().Bool("json", false, "output as JSON")
}
