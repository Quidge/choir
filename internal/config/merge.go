package config

import "fmt"

// FlagOverrides contains CLI flag values that override configuration.
type FlagOverrides struct {
	Backend string
	CPUs    int
	Memory  string
	Disk    string
}

// Merge combines global config, project config, and CLI flag overrides
// following the precedence order: backend defaults → global → project → flags.
// Returns the merged configuration ready for use.
func Merge(global GlobalConfig, project ProjectConfig, flags FlagOverrides) (MergedConfig, error) {
	merged := MergedConfig{}

	// Determine which backend to use
	merged.Backend = global.DefaultBackend
	if flags.Backend != "" {
		merged.Backend = flags.Backend
	}

	// Get backend configuration
	backend, ok := global.Backends[merged.Backend]
	if !ok {
		return MergedConfig{}, fmt.Errorf("unknown backend: %s", merged.Backend)
	}
	merged.BackendType = backend.Type

	// Merge resources: backend defaults → project config → flags
	merged.Resources = Resources{
		CPUs:   backend.CPUs,
		Memory: backend.Memory,
		Disk:   backend.Disk,
	}

	// Project config overrides backend defaults
	if project.Resources.CPUs != 0 {
		merged.Resources.CPUs = project.Resources.CPUs
	}
	if project.Resources.Memory != "" {
		merged.Resources.Memory = project.Resources.Memory
	}
	if project.Resources.Disk != "" {
		merged.Resources.Disk = project.Resources.Disk
	}

	// CLI flags override everything
	if flags.CPUs != 0 {
		merged.Resources.CPUs = flags.CPUs
	}
	if flags.Memory != "" {
		merged.Resources.Memory = flags.Memory
	}
	if flags.Disk != "" {
		merged.Resources.Disk = flags.Disk
	}

	// Expand credentials from global config
	expandedCreds, err := ExpandCredentials(global.Credentials)
	if err != nil {
		return MergedConfig{}, fmt.Errorf("failed to expand credentials: %w", err)
	}
	merged.Credentials = expandedCreds

	// Copy project-specific settings
	merged.BaseImage = project.BaseImage
	merged.Packages = project.Packages
	merged.Setup = project.Setup
	merged.BranchPrefix = project.BranchPrefix

	// Expand environment variables
	if project.Env != nil {
		expandedEnv, err := ExpandEnvMap(project.Env)
		if err != nil {
			return MergedConfig{}, fmt.Errorf("failed to expand environment variables: %w", err)
		}
		merged.Env = expandedEnv
	}

	// Expand file mount source paths
	if project.Files != nil {
		expandedFiles, err := ExpandFileMounts(project.Files)
		if err != nil {
			return MergedConfig{}, fmt.Errorf("failed to expand file mounts: %w", err)
		}
		merged.Files = expandedFiles
	}

	return merged, nil
}

// Load loads both global and project configuration, then merges them
// with the provided flag overrides.
func Load(projectDir string, flags FlagOverrides) (MergedConfig, error) {
	global, err := LoadGlobalConfig()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("failed to load global config: %w", err)
	}

	project, err := LoadProjectConfigFromDir(projectDir)
	if err != nil {
		return MergedConfig{}, fmt.Errorf("failed to load project config: %w", err)
	}

	return Merge(global, project, flags)
}

// LoadFromCwd loads configuration using the current working directory
// as the project directory.
func LoadFromCwd(flags FlagOverrides) (MergedConfig, error) {
	global, err := LoadGlobalConfig()
	if err != nil {
		return MergedConfig{}, fmt.Errorf("failed to load global config: %w", err)
	}

	project, err := LoadProjectConfig("")
	if err != nil {
		return MergedConfig{}, fmt.Errorf("failed to load project config: %w", err)
	}

	return Merge(global, project, flags)
}
