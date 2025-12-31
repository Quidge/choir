package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR} or ${VAR:-default} patterns.
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return home + path[1:], nil
	}

	if path == "~" {
		return os.UserHomeDir()
	}

	return path, nil
}

// ExpandEnvVars expands ${VAR} patterns in a string using environment variables.
// If a variable is not set, it expands to an empty string.
func ExpandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := match[2 : len(match)-1]

		// Handle default values: ${VAR:-default}
		if idx := strings.Index(varName, ":-"); idx != -1 {
			name := varName[:idx]
			defaultVal := varName[idx+2:]
			if val, ok := os.LookupEnv(name); ok {
				return val
			}
			return defaultVal
		}

		return os.Getenv(varName)
	})
}

// ReadFromFile reads the contents of a file and returns it as a string.
// The path is first expanded (~ expansion) before reading.
func ReadFromFile(path string) (string, error) {
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Trim trailing newlines (common in secret files)
	return strings.TrimRight(string(data), "\n\r"), nil
}

// ExpandEnvMap processes a map of EnvVar values, expanding environment
// variables and reading from_file references. Returns a map of string values.
func ExpandEnvMap(envVars map[string]EnvVar) (map[string]string, error) {
	result := make(map[string]string, len(envVars))

	for key, envVar := range envVars {
		var value string
		var err error

		if envVar.FromFile != "" {
			// Expand path first (in case it contains ~)
			expandedPath := ExpandEnvVars(envVar.FromFile)
			value, err = ReadFromFile(expandedPath)
			if err != nil {
				return nil, fmt.Errorf("failed to expand env var %s: %w", key, err)
			}
		} else {
			// Expand environment variables in the value
			value = ExpandEnvVars(envVar.Value)
		}

		result[key] = value
	}

	return result, nil
}

// ExpandCredentials expands all paths in a CredentialsConfig.
func ExpandCredentials(creds CredentialsConfig) (CredentialsConfig, error) {
	var err error
	expanded := CredentialsConfig{}

	expanded.ClaudeConfig, err = ExpandPath(creds.ClaudeConfig)
	if err != nil {
		return expanded, fmt.Errorf("claude_config: %w", err)
	}

	expanded.SSHKeys, err = ExpandPath(creds.SSHKeys)
	if err != nil {
		return expanded, fmt.Errorf("ssh_keys: %w", err)
	}

	expanded.GitConfig, err = ExpandPath(creds.GitConfig)
	if err != nil {
		return expanded, fmt.Errorf("git_config: %w", err)
	}

	expanded.GitHubCLI, err = ExpandPath(creds.GitHubCLI)
	if err != nil {
		return expanded, fmt.Errorf("github_cli: %w", err)
	}

	return expanded, nil
}

// ExpandFileMounts expands source paths in file mounts.
// Relative source paths are resolved relative to baseDir (the directory
// containing the project config file).
func ExpandFileMounts(files []FileMount, baseDir string) ([]FileMount, error) {
	result := make([]FileMount, len(files))
	for i, f := range files {
		// First expand tilde
		expandedSource, err := ExpandPath(f.Source)
		if err != nil {
			return nil, fmt.Errorf("file mount %d source: %w", i, err)
		}
		// Then resolve relative paths against baseDir
		if baseDir != "" && !filepath.IsAbs(expandedSource) {
			expandedSource = filepath.Clean(filepath.Join(baseDir, expandedSource))
		}
		result[i] = FileMount{
			Source:   expandedSource,
			Target:   f.Target,
			ReadOnly: f.ReadOnly,
		}
	}
	return result, nil
}
