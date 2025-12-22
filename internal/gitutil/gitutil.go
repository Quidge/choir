// Package gitutil provides utilities for git operations.
// It uses os/exec to call git commands rather than git libraries.
package gitutil

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	// ErrNotGitRepo is returned when the directory is not inside a git repository.
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrDetachedHead is returned when the repository is in detached HEAD state.
	ErrDetachedHead = errors.New("repository is in detached HEAD state")

	// ErrNoRemote is returned when no remote URL is configured.
	ErrNoRemote = errors.New("no remote URL configured")
)

// RepoRoot returns the root directory of the git repository containing dir.
// If dir is empty, the current working directory is used.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrNotGitRepo
		}
		return "", fmt.Errorf("failed to get repo root: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// RemoteURL returns the URL of the specified remote (typically "origin").
// If remoteName is empty, "origin" is used.
// If dir is empty, the current working directory is used.
func RemoteURL(dir, remoteName string) (string, error) {
	if remoteName == "" {
		remoteName = "origin"
	}

	cmd := exec.Command("git", "remote", "get-url", remoteName)
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", ErrNoRemote
		}
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the name of the current branch.
// Returns ErrDetachedHead if the repository is in detached HEAD state.
// If dir is empty, the current working directory is used.
func CurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Check if it's a detached HEAD
			if IsDetachedHead(dir) {
				return "", ErrDetachedHead
			}
			return "", ErrNotGitRepo
		}
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// IsDetachedHead returns true if the repository is in detached HEAD state.
// If dir is empty, the current working directory is used.
func IsDetachedHead(dir string) bool {
	cmd := exec.Command("git", "symbolic-ref", "-q", "HEAD")
	if dir != "" {
		cmd.Dir = dir
	}

	err := cmd.Run()
	// symbolic-ref returns non-zero exit code if HEAD is not a symbolic ref (i.e., detached)
	return err != nil
}

// IsValidBranchName checks if a string is a valid git branch name.
// This is a simplified check that validates common branch name characters
// suitable for use in task IDs.
func IsValidBranchName(name string) bool {
	if name == "" {
		return false
	}

	// Check for invalid patterns
	if strings.Contains(name, "..") {
		return false
	}
	if strings.Contains(name, "//") {
		return false
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return false
	}
	if strings.HasSuffix(name, ".lock") {
		return false
	}
	if strings.Contains(name, "@{") {
		return false
	}

	// Check for control characters and invalid chars
	for _, c := range name {
		if c < 32 || c == 127 { // control characters
			return false
		}
		switch c {
		case ' ', '~', '^', ':', '?', '*', '[', '\\':
			return false
		}
	}

	return true
}

// ValidateBranchName returns an error if the name is not a valid git branch name.
func ValidateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name is empty")
	}
	if !IsValidBranchName(name) {
		return fmt.Errorf("invalid branch name: %s", name)
	}
	return nil
}

// IsInsideWorkTree returns true if dir is inside a git work tree.
// If dir is empty, the current working directory is used.
func IsInsideWorkTree(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if dir != "" {
		cmd.Dir = dir
	}

	out, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(out)) == "true"
}
