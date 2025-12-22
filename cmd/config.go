package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify global configuration",
	Long: `View or modify the global choir configuration.

Subcommands:
  show   Print current configuration
  edit   Open configuration in $EDITOR
  set    Set a specific configuration key`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("config show not implemented")
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in $EDITOR",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("config edit not implemented")
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration key",
	Long: `Set a specific configuration key using dot notation.

Example:
  choir config set backends.local.memory 8GB`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		return fmt.Errorf("config set not implemented: %s=%s", key, value)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configSetCmd)
}
