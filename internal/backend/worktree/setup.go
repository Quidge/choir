package worktree

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// HostSetupRunner implements backend.SetupRunner for the worktree backend.
// It executes setup steps directly on the host filesystem.
type HostSetupRunner struct {
	// WorkDir is the worktree directory where setup runs.
	WorkDir string
}

// Ensure HostSetupRunner implements SetupRunner.
var _ backend.SetupRunner = (*HostSetupRunner)(nil)

// Run executes all setup steps for the worktree.
//
// Setup order:
// 1. Write environment variables to .choir-env file
// 2. Create symlinks or copy files
// 3. Run setup commands
func (r *HostSetupRunner) Run(ctx context.Context, cfg *backend.SetupConfig) error {
	if r.WorkDir == "" {
		return fmt.Errorf("work directory not set")
	}

	// Check context before each step
	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 1: Write environment to .choir-env file
	if err := r.writeEnvironment(cfg.Environment); err != nil {
		return fmt.Errorf("failed to write environment: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 2: Handle file mounts (symlinks or copies)
	if err := r.handleFiles(cfg.Files); err != nil {
		return fmt.Errorf("failed to handle files: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 3: Run setup commands
	if err := r.runCommands(ctx, cfg.SetupCommands); err != nil {
		return fmt.Errorf("failed to run setup commands: %w", err)
	}

	return nil
}

// writeEnvironment writes environment variables to the .choir-env file.
// The file is written in a format that can be sourced by shell.
func (r *HostSetupRunner) writeEnvironment(env map[string]string) error {
	if len(env) == 0 {
		return nil
	}

	envPath := filepath.Join(r.WorkDir, envFile)
	f, err := os.Create(envPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write header
	if _, err := f.WriteString("# Choir environment variables\n"); err != nil {
		return err
	}
	if _, err := f.WriteString("# This file is auto-generated. Do not edit manually.\n\n"); err != nil {
		return err
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Write each variable as export statement
	for _, key := range keys {
		value := env[key]
		// Escape single quotes in value for shell safety
		escapedValue := strings.ReplaceAll(value, "'", "'\\''")
		line := fmt.Sprintf("export %s='%s'\n", key, escapedValue)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// handleFiles processes file mounts by creating symlinks or copying files.
func (r *HostSetupRunner) handleFiles(files []config.FileMount) error {
	for _, fm := range files {
		if err := r.handleFile(fm); err != nil {
			return fmt.Errorf("failed to handle file %s: %w", fm.Source, err)
		}
	}
	return nil
}

// handleFile processes a single file mount.
// Uses symlinks when possible (preferred), copies when necessary.
func (r *HostSetupRunner) handleFile(fm config.FileMount) error {
	source := fm.Source
	target := fm.Target

	// If target is relative, make it relative to worktree
	if !filepath.IsAbs(target) {
		target = filepath.Join(r.WorkDir, target)
	}

	// Check if source exists
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("source not found: %w", err)
	}

	// Create parent directory for target if needed
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Remove existing target if present
	if _, err := os.Lstat(target); err == nil {
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("failed to remove existing target: %w", err)
		}
	}

	// Determine whether to symlink or copy
	// Prefer symlink for readonly mounts (saves disk space)
	// Copy for non-readonly mounts or if source is outside the main repo
	if fm.ReadOnly {
		// Use symlink
		if err := os.Symlink(source, target); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	} else {
		// Copy the file or directory
		if sourceInfo.IsDir() {
			if err := copyDir(source, target); err != nil {
				return err
			}
		} else {
			if err := copyFile(source, target); err != nil {
				return err
			}
		}
	}

	return nil
}

// runCommands executes setup commands in the worktree directory.
func (r *HostSetupRunner) runCommands(ctx context.Context, commands []string) error {
	if len(commands) == 0 {
		return nil
	}

	shell, err := validShell()
	if err != nil {
		return err
	}

	envPath := filepath.Join(r.WorkDir, envFile)

	for i, command := range commands {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Build command that sources env file first
		var fullCmd string
		if _, err := os.Stat(envPath); err == nil {
			fullCmd = fmt.Sprintf("source %q && %s", envPath, command)
		} else {
			fullCmd = command
		}

		cmd := exec.CommandContext(ctx, shell, "-c", fullCmd)
		cmd.Dir = r.WorkDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("command %d failed: %s: %w", i+1, command, err)
		}
	}

	return nil
}

// copyFile copies a single file from src to dst using streaming to handle large files.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	srcInfo, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies a directory from src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
