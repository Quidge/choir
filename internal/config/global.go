package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfigPath returns the path to the global configuration file.
func GlobalConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}
	return filepath.Join(configDir, "choir", "config.yaml"), nil
}

// LoadGlobalConfig loads the global configuration from ~/.config/choir/config.yaml.
// If the file doesn't exist, returns default configuration (not an error).
// If the file exists but is invalid YAML, returns an error.
func LoadGlobalConfig() (GlobalConfig, error) {
	configPath, err := GlobalConfigPath()
	if err != nil {
		return DefaultGlobalConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultGlobalConfig(), nil
		}
		return GlobalConfig{}, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return GlobalConfig{}, fmt.Errorf("invalid YAML in %s: %w", configPath, err)
	}

	// Apply defaults for missing fields
	cfg = applyGlobalDefaults(cfg)

	return cfg, nil
}

// applyGlobalDefaults fills in missing fields with default values.
func applyGlobalDefaults(cfg GlobalConfig) GlobalConfig {
	defaults := DefaultGlobalConfig()

	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}
	if cfg.DefaultBackend == "" {
		cfg.DefaultBackend = defaults.DefaultBackend
	}

	// Apply credential defaults
	if cfg.Credentials.ClaudeConfig == "" {
		cfg.Credentials.ClaudeConfig = defaults.Credentials.ClaudeConfig
	}
	if cfg.Credentials.SSHKeys == "" {
		cfg.Credentials.SSHKeys = defaults.Credentials.SSHKeys
	}
	if cfg.Credentials.GitConfig == "" {
		cfg.Credentials.GitConfig = defaults.Credentials.GitConfig
	}
	if cfg.Credentials.GitHubCLI == "" {
		cfg.Credentials.GitHubCLI = defaults.Credentials.GitHubCLI
	}

	// Ensure backends map exists
	if cfg.Backends == nil {
		cfg.Backends = defaults.Backends
	} else {
		// Apply defaults to local backend if it exists but has missing fields
		if local, ok := cfg.Backends["local"]; ok {
			defaultLocal := defaults.Backends["local"]
			if local.Type == "" {
				local.Type = defaultLocal.Type
			}
			if local.CPUs == 0 {
				local.CPUs = defaultLocal.CPUs
			}
			if local.Memory == "" {
				local.Memory = defaultLocal.Memory
			}
			if local.Disk == "" {
				local.Disk = defaultLocal.Disk
			}
			if local.VMType == "" {
				local.VMType = defaultLocal.VMType
			}
			cfg.Backends["local"] = local
		}
	}

	return cfg
}

// EnsureGlobalConfigDir creates the global config directory if it doesn't exist.
func EnsureGlobalConfigDir() error {
	configPath, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(configPath), 0755)
}

// WriteGlobalConfig writes a global configuration to the default path.
func WriteGlobalConfig(cfg GlobalConfig) error {
	configPath, err := GlobalConfigPath()
	if err != nil {
		return err
	}

	if err := EnsureGlobalConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
