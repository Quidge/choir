package config

import (
	"fmt"
)

// ValidateFileMounts validates file mounts.
// Source paths are expected to be already expanded by ExpandFileMounts.
// Target paths can be absolute or relative (relative paths are resolved
// by backends relative to the workspace root).
func ValidateFileMounts(files []FileMount) error {
	for i, f := range files {
		if f.Target == "" {
			return fmt.Errorf("file mount %d: target path is required", i)
		}
	}
	return nil
}

// NewCreateConfig builds a CreateConfig from a MergedConfig, repository info, and environment ID.
// It performs final validation including target path checks.
func NewCreateConfig(merged MergedConfig, repo RepositoryInfo, id string) (CreateConfig, error) {
	if id == "" {
		return CreateConfig{}, fmt.Errorf("environment ID is required")
	}

	if repo.Path == "" {
		return CreateConfig{}, fmt.Errorf("repository path is required")
	}

	// Validate file mount target paths
	if err := ValidateFileMounts(merged.Files); err != nil {
		return CreateConfig{}, fmt.Errorf("invalid file mounts: %w", err)
	}

	return CreateConfig{
		ID:            id,
		Backend:       merged.Backend,
		BackendType:   merged.BackendType,
		Resources:     merged.Resources,
		Credentials:   merged.Credentials,
		Repository:    repo,
		BaseImage:     merged.BaseImage,
		Packages:      merged.Packages,
		Environment:   merged.Env,
		Files:         merged.Files,
		SetupCommands: merged.Setup,
		BranchPrefix:  merged.BranchPrefix,
	}, nil
}
