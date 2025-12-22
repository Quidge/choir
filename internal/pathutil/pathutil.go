// Package pathutil provides utilities for path resolution and validation.
package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandTilde expands ~ to the user's home directory.
// Returns the path unchanged if it doesn't start with ~.
func ExpandTilde(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path == "~" {
		return os.UserHomeDir()
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

// ResolveRelative resolves a path relative to a base directory.
// If path is absolute, it is returned unchanged (after cleaning).
// If path is relative, it is joined with base and cleaned.
func ResolveRelative(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

// IsAbsolute returns true if the path is an absolute path.
func IsAbsolute(path string) bool {
	return filepath.IsAbs(path)
}

// ValidateAbsolute checks if a path is absolute and returns an error if not.
func ValidateAbsolute(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute: %s", path)
	}
	return nil
}

// Exists returns true if the path exists on the filesystem.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExistsAndIsDir returns true if the path exists and is a directory.
func ExistsAndIsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ExistsAndIsFile returns true if the path exists and is a regular file.
func ExistsAndIsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
