package config

import (
	"fmt"

	"github.com/Quidge/choir/internal/pathutil"
)

// ValidateFileMounts validates that all file mount target paths are absolute.
// Source paths are expected to be already expanded by ExpandFileMounts.
func ValidateFileMounts(files []FileMount) error {
	for i, f := range files {
		if err := pathutil.ValidateAbsolute(f.Target); err != nil {
			return fmt.Errorf("file mount %d target: %w", i, err)
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
