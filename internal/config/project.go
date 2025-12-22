package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfigFilename is the name of the project configuration file.
const ProjectConfigFilename = ".choir.yaml"

// FindProjectConfig searches for a .choir.yaml file starting from the given
// directory and walking up to parent directories until it finds one or reaches
// the filesystem root.
func FindProjectConfig(startDir string) (string, error) {
	dir := startDir
	for {
		configPath := filepath.Join(dir, ProjectConfigFilename)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// LoadProjectConfig loads the project configuration from .choir.yaml.
// If configPath is empty, searches from the current directory.
// If the file doesn't exist, returns default configuration (not an error).
// If the file exists but is invalid YAML, returns an error.
func LoadProjectConfig(configPath string) (ProjectConfig, error) {
	if configPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return DefaultProjectConfig(), nil
		}
		configPath, err = FindProjectConfig(cwd)
		if err != nil {
			return DefaultProjectConfig(), nil
		}
		if configPath == "" {
			return DefaultProjectConfig(), nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultProjectConfig(), nil
		}
		return ProjectConfig{}, fmt.Errorf("failed to read project config: %w", err)
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("invalid YAML in %s: %w", configPath, err)
	}

	// Apply defaults for missing fields
	cfg = applyProjectDefaults(cfg)

	return cfg, nil
}

// LoadProjectConfigFromDir loads the project configuration from a specific directory.
func LoadProjectConfigFromDir(dir string) (ProjectConfig, error) {
	configPath := filepath.Join(dir, ProjectConfigFilename)
	return LoadProjectConfig(configPath)
}

// applyProjectDefaults fills in missing fields with default values.
func applyProjectDefaults(cfg ProjectConfig) ProjectConfig {
	defaults := DefaultProjectConfig()

	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}
	if cfg.BranchPrefix == "" {
		cfg.BranchPrefix = defaults.BranchPrefix
	}

	return cfg
}

// WriteProjectConfig writes a project configuration to the specified path.
func WriteProjectConfig(configPath string, cfg ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ProjectConfigExists checks if a .choir.yaml file exists in the given directory.
func ProjectConfigExists(dir string) bool {
	configPath := filepath.Join(dir, ProjectConfigFilename)
	_, err := os.Stat(configPath)
	return err == nil
}
