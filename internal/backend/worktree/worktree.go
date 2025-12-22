// Package worktree implements the worktree backend for choir.
// This backend creates isolated workspaces using git worktrees instead of VMs.
//
// Key characteristics:
//   - No process/network isolation (all agents share host environment)
//   - Fast creation (just git worktree add)
//   - Shares host credentials (no copying needed)
//   - Worktrees created at: <repo-parent>/choir-<task-id>/
package worktree

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// cleanGitEnv returns a clean environment without git-specific variables
// that might interfere with git operations (e.g., when running inside git hooks).
func cleanGitEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_") {
			env = append(env, e)
		}
	}
	return env
}

const (
	// BackendType is the identifier for this backend type.
	BackendType = "worktree"

	// markerFile is the file created in each worktree to identify it as choir-managed.
	markerFile = ".choir-agent"

	// envFile is the file where environment variables are stored.
	envFile = ".choir-env"

	// worktreePrefix is the directory prefix for choir worktrees.
	worktreePrefix = "choir-"
)

// Backend implements the backend.Backend interface using git worktrees.
type Backend struct {
	// repoRoot is the root of the main git repository.
	// This is determined dynamically based on the CreateConfig.
	repoRoot string
}

// New creates a new worktree backend.
func New(cfg backend.BackendConfig) (backend.Backend, error) {
	return &Backend{}, nil
}

func init() {
	backend.Register(BackendType, New)
}

// Create provisions a new workspace using git worktree.
// The backendID returned is the absolute path to the worktree directory.
func (b *Backend) Create(ctx context.Context, cfg *config.CreateConfig) (string, error) {
	if cfg.TaskID == "" {
		return "", errors.New("task ID is required")
	}

	if cfg.Repository.Path == "" {
		return "", errors.New("repository path is required")
	}

	// Warn if packages are specified (worktree backend can't install them)
	if len(cfg.Packages) > 0 {
		fmt.Fprintf(os.Stderr, "warning: worktree backend ignores packages configuration\n")
	}

	repoRoot := cfg.Repository.Path
	b.repoRoot = repoRoot

	// Determine worktree location: <repo-parent>/choir-<task-id>/
	repoParent := filepath.Dir(repoRoot)
	worktreePath := filepath.Join(repoParent, worktreePrefix+cfg.TaskID)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree already exists: %s", worktreePath)
	}

	// Determine branch name
	branchName := cfg.BranchPrefix + cfg.TaskID
	if cfg.BranchPrefix == "" {
		branchName = "agent/" + cfg.TaskID
	}

	// Determine base branch
	baseBranch := cfg.Repository.BaseBranch
	if baseBranch == "" {
		baseBranch = "HEAD"
	}

	// Create the worktree with a new branch
	// git worktree add -b <branch> <path> <base>
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = repoRoot
	cmd.Env = cleanGitEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w\noutput: %s", err, output)
	}

	// Create the marker file to identify this as a choir-managed worktree
	markerPath := filepath.Join(worktreePath, markerFile)
	markerContent := fmt.Sprintf("task_id: %s\ncreated_by: choir\n", cfg.TaskID)
	if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
		// Try to clean up the worktree on failure
		_ = b.Destroy(ctx, worktreePath)
		return "", fmt.Errorf("failed to create marker file: %w", err)
	}

	return worktreePath, nil
}

// NewSetupRunner returns a HostSetupRunner for this worktree.
func (b *Backend) NewSetupRunner(backendID string) backend.SetupRunner {
	return &HostSetupRunner{
		WorkDir: backendID,
	}
}

// Start is a no-op for worktrees (they are always available).
func (b *Backend) Start(ctx context.Context, backendID string) error {
	// Verify the worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", backendID)
	}
	return nil
}

// Stop is a no-op for worktrees.
func (b *Backend) Stop(ctx context.Context, backendID string) error {
	// Verify the worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", backendID)
	}
	return nil
}

// Destroy removes a worktree using git worktree remove.
func (b *Backend) Destroy(ctx context.Context, backendID string) error {
	// Find the main repo root by checking git config
	repoRoot, err := findMainRepo(backendID)
	if err != nil {
		// If we can't find the main repo, try direct removal
		return os.RemoveAll(backendID)
	}

	// Use git worktree remove --force
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", backendID)
	cmd.Dir = repoRoot
	cmd.Env = cleanGitEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If git worktree remove fails, fall back to manual removal
		if rmErr := os.RemoveAll(backendID); rmErr != nil {
			return fmt.Errorf("failed to remove worktree: %w\ngit output: %s\nmanual removal error: %v", err, output, rmErr)
		}
	}

	return nil
}

// Shell opens an interactive shell in the worktree directory.
// It sources the .choir-env file if present.
func (b *Backend) Shell(ctx context.Context, backendID string) error {
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		return fmt.Errorf("worktree not found: %s", backendID)
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	// Build the command to source env file if it exists, then exec shell
	envPath := filepath.Join(backendID, envFile)
	var cmd *exec.Cmd
	if _, err := os.Stat(envPath); err == nil {
		// Source the env file before starting the shell
		cmd = exec.CommandContext(ctx, shell, "-c", fmt.Sprintf("source %q && exec %s", envPath, shell))
	} else {
		cmd = exec.CommandContext(ctx, shell)
	}

	cmd.Dir = backendID
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up process group for signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return cmd.Run()
}

// Exec runs a command in the worktree directory and returns output.
func (b *Backend) Exec(ctx context.Context, backendID string, command string) (string, int, error) {
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		return "", -1, fmt.Errorf("worktree not found: %s", backendID)
	}

	// Build the shell command, sourcing env file if present
	envPath := filepath.Join(backendID, envFile)
	var shellCmd string
	if _, err := os.Stat(envPath); err == nil {
		shellCmd = fmt.Sprintf("source %q && %s", envPath, command)
	} else {
		shellCmd = command
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.CommandContext(ctx, shell, "-c", shellCmd)
	cmd.Dir = backendID

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return string(output), -1, err
		}
	}

	return string(output), exitCode, nil
}

// Status returns the current status of a worktree.
func (b *Backend) Status(ctx context.Context, backendID string) (backend.BackendStatus, error) {
	info, err := os.Stat(backendID)
	if os.IsNotExist(err) {
		return backend.BackendStatus{
			State:   backend.StateNotFound,
			Message: "worktree directory does not exist",
		}, nil
	}
	if err != nil {
		return backend.BackendStatus{
			State:   backend.StateError,
			Message: fmt.Sprintf("failed to stat worktree: %v", err),
		}, nil
	}

	if !info.IsDir() {
		return backend.BackendStatus{
			State:   backend.StateError,
			Message: "path exists but is not a directory",
		}, nil
	}

	// Check for marker file to confirm it's a choir worktree
	markerPath := filepath.Join(backendID, markerFile)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return backend.BackendStatus{
			State:   backend.StateError,
			Message: "directory exists but is not a choir-managed worktree",
		}, nil
	}

	return backend.BackendStatus{
		State:   backend.StateRunning,
		Message: "worktree is ready",
	}, nil
}

// List returns all choir-managed worktrees.
func (b *Backend) List(ctx context.Context) ([]string, error) {
	// We need a repo root to list worktrees
	// Try to find it from current directory
	repoRoot := b.repoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = findRepoRoot("")
		if err != nil {
			return nil, fmt.Errorf("cannot list worktrees: not in a git repository")
		}
	}

	// Parse git worktree list --porcelain
	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = repoRoot
	cmd.Env = cleanGitEnv()
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var choirWorktrees []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentWorktree string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "worktree ") {
			currentWorktree = strings.TrimPrefix(line, "worktree ")
		} else if line == "" {
			// End of a worktree entry
			if currentWorktree != "" && isChoirManaged(currentWorktree) {
				choirWorktrees = append(choirWorktrees, currentWorktree)
			}
			currentWorktree = ""
		}
	}

	// Handle last entry if no trailing newline
	if currentWorktree != "" && isChoirManaged(currentWorktree) {
		choirWorktrees = append(choirWorktrees, currentWorktree)
	}

	return choirWorktrees, nil
}

// isChoirManaged checks if a worktree directory is managed by choir.
// A worktree is choir-managed if:
// 1. Its directory name starts with "choir-"
// 2. It contains a .choir-agent marker file
func isChoirManaged(worktreePath string) bool {
	// Check naming convention
	dirName := filepath.Base(worktreePath)
	if !strings.HasPrefix(dirName, worktreePrefix) {
		return false
	}

	// Check for marker file
	markerPath := filepath.Join(worktreePath, markerFile)
	_, err := os.Stat(markerPath)
	return err == nil
}

// findMainRepo finds the main repository root from a worktree path.
func findMainRepo(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = worktreePath
	cmd.Env = cleanGitEnv()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	gitCommonDir := strings.TrimSpace(string(output))
	// git-common-dir returns the .git directory of the main repo
	// We need the parent of that
	if filepath.IsAbs(gitCommonDir) {
		return filepath.Dir(gitCommonDir), nil
	}
	// If relative, it's relative to worktreePath
	absGitDir := filepath.Join(worktreePath, gitCommonDir)
	return filepath.Dir(absGitDir), nil
}

// findRepoRoot finds the repository root from a directory.
func findRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = cleanGitEnv()
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
