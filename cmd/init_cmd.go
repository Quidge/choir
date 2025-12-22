package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quidge/choir/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a .choir.yaml template",
	Long: `Create a .choir.yaml template in the current directory.

The template includes commented examples for all configuration options.`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().Bool("force", false, "overwrite existing file")
}

func runInit(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath := filepath.Join(cwd, config.ProjectConfigFilename)

	// Check if file already exists
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s already exists (use --force to overwrite)", config.ProjectConfigFilename)
		}
	}

	// Write the template
	if err := os.WriteFile(configPath, []byte(config.ProjectConfigTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", config.ProjectConfigFilename, err)
	}

	fmt.Printf("Created %s\n", config.ProjectConfigFilename)
	return nil
}
