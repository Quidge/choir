package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Quidge/choir/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	RunE:  runConfigShow,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in $EDITOR",
	Args:  cobra.NoArgs,
	RunE:  runConfigEdit,
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration key",
	Long: `Set a specific configuration key using dot notation.

Example:
  choir config set backends.local.memory 8GB`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadGlobalConfig()
	if err != nil {
		return err
	}

	configPath, _ := config.GlobalConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("# No config file found at %s\n", configPath)
		fmt.Println("# Showing default configuration:")
		fmt.Println()
	} else {
		fmt.Printf("# Config file: %s\n\n", configPath)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Print(string(data))
	return nil
}

func runConfigEdit(_ *cobra.Command, _ []string) error {
	configPath, err := config.GlobalConfigPath()
	if err != nil {
		return err
	}

	// Create config file with template if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := config.EnsureGlobalConfigDir(); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		if err := os.WriteFile(configPath, []byte(config.GlobalConfigTemplate), 0644); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, configPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// For now, return not implemented - proper implementation would require
	// reflection or a more sophisticated approach to set nested YAML values
	return fmt.Errorf("config set not implemented: %s=%s\nPlease use 'choir config edit' instead", key, value)
}
