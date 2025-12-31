package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree"
	"github.com/Quidge/choir/internal/config"
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

// cleanGitEnv returns a clean environment without git-specific variables
// that might interfere with git operations.
func cleanGitEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_") {
			env = append(env, e)
		}
	}
	return env
}

// setupTestRepo creates a temporary git repository for testing.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	env := cleanGitEnv()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init repo: %v\n%s", err, out)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	cmd.Env = env
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	cmd.Env = env
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Env = env
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to commit: %v\n%s", err, out)
	}

	return repoDir
}

// openTestDB creates an in-memory database for testing.
func openTestDB(t *testing.T) *state.DB {
	t.Helper()
	db, err := state.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestEnvCreateIntegration tests the env create flow with real git repo and database.
func TestEnvCreateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := setupTestRepo(t)

	// Create a temporary database
	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Get backend
	be, err := backend.Get(backend.BackendConfig{
		Name: "local",
		Type: "worktree",
	})
	if err != nil {
		t.Fatalf("failed to get backend: %v", err)
	}

	ctx := context.Background()
	envID, err := state.GenerateID()
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}
	shortID := state.ShortID(envID)

	// Build config manually (simulating what env create does)
	createCfg := &config.CreateConfig{
		ID:           envID,
		Backend:      "local",
		BackendType:  "worktree",
		BranchPrefix: "env/",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	// Create environment record
	env := &state.Environment{
		ID:         envID,
		Backend:    "local",
		RepoPath:   repoDir,
		BranchName: "env/" + shortID,
		BaseBranch: "HEAD",
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("failed to create environment: %v", err)
	}

	// Create worktree
	backendID, err := be.Create(ctx, createCfg)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer be.Destroy(ctx, backendID)

	// Update environment
	env.BackendID = backendID
	env.Status = state.StatusReady
	if err := db.UpdateEnvironment(env); err != nil {
		t.Fatalf("failed to update environment: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Error("worktree was not created")
	}

	// Verify environment in database
	gotEnv, err := db.GetEnvironment(envID)
	if err != nil {
		t.Fatalf("failed to get environment: %v", err)
	}

	if gotEnv.Status != state.StatusReady {
		t.Errorf("expected status ready, got %s", gotEnv.Status)
	}

	if gotEnv.BackendID != backendID {
		t.Errorf("expected backendID %s, got %s", backendID, gotEnv.BackendID)
	}
}

// TestEnvAttachCommand tests the attach command logic.
func TestEnvAttachCommand(t *testing.T) {
	db := openTestDB(t)

	t.Run("environment not found", func(t *testing.T) {
		_, err := db.GetEnvironmentByPrefix("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent environment")
		}
	})

	t.Run("environment removed status", func(t *testing.T) {
		env := &state.Environment{
			ID:         "removed123456789012345678901234",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "env/removed",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     state.StatusRemoved,
		}
		if err := db.CreateEnvironment(env); err != nil {
			t.Fatalf("failed to create environment: %v", err)
		}

		gotEnv, err := db.GetEnvironment("removed123456789012345678901234")
		if err != nil {
			t.Fatalf("failed to get environment: %v", err)
		}

		if gotEnv.Status != state.StatusRemoved {
			t.Errorf("expected status removed, got %s", gotEnv.Status)
		}
	})

	t.Run("environment failed status", func(t *testing.T) {
		env := &state.Environment{
			ID:         "failed1234567890123456789012345",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "env/failed",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     state.StatusFailed,
		}
		if err := db.CreateEnvironment(env); err != nil {
			t.Fatalf("failed to create environment: %v", err)
		}

		gotEnv, err := db.GetEnvironment("failed1234567890123456789012345")
		if err != nil {
			t.Fatalf("failed to get environment: %v", err)
		}

		if gotEnv.Status != state.StatusFailed {
			t.Errorf("expected status failed, got %s", gotEnv.Status)
		}
	})
}

// TestEnvListCommand tests the list command logic.
func TestEnvListCommand(t *testing.T) {
	db := openTestDB(t)

	t.Run("empty list", func(t *testing.T) {
		envs, err := db.ListEnvironments(state.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list environments: %v", err)
		}

		if len(envs) != 0 {
			t.Errorf("expected 0 environments, got %d", len(envs))
		}
	})

	t.Run("list with environments", func(t *testing.T) {
		// Create test environments
		for i, status := range []state.EnvironmentStatus{state.StatusReady, state.StatusProvisioning, state.StatusFailed} {
			env := &state.Environment{
				ID:         string(rune('a'+i)) + "bc123456789012345678901234567",
				Backend:    "local",
				RepoPath:   "/test",
				BranchName: "env/test",
				BaseBranch: "main",
				CreatedAt:  time.Now(),
				Status:     status,
			}
			if err := db.CreateEnvironment(env); err != nil {
				t.Fatalf("failed to create environment: %v", err)
			}
		}

		// List all environments
		envs, err := db.ListEnvironments(state.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list environments: %v", err)
		}

		if len(envs) != 3 {
			t.Errorf("expected 3 environments, got %d", len(envs))
		}

		// List only ready/provisioning environments
		envs, err = db.ListEnvironments(state.ListOptions{
			Statuses: []state.EnvironmentStatus{state.StatusReady, state.StatusProvisioning},
		})
		if err != nil {
			t.Fatalf("failed to list environments: %v", err)
		}

		if len(envs) != 2 {
			t.Errorf("expected 2 active environments, got %d", len(envs))
		}
	})
}

// TestEnvRmCommand tests the rm command logic.
func TestEnvRmCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	repoDir := setupTestRepo(t)
	ctx := context.Background()

	// Create database
	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Get backend
	be, err := backend.Get(backend.BackendConfig{
		Name: "local",
		Type: "worktree",
	})
	if err != nil {
		t.Fatalf("failed to get backend: %v", err)
	}

	envID, _ := state.GenerateID()
	shortID := state.ShortID(envID)

	// Create worktree
	createCfg := &config.CreateConfig{
		ID:           envID,
		Backend:      "local",
		BackendType:  "worktree",
		BranchPrefix: "env/",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := be.Create(ctx, createCfg)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create environment record
	env := &state.Environment{
		ID:         envID,
		Backend:    "local",
		BackendID:  backendID,
		RepoPath:   repoDir,
		BranchName: "env/" + shortID,
		BaseBranch: "HEAD",
		CreatedAt:  time.Now(),
		Status:     state.StatusReady,
	}

	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("failed to create environment: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Destroy worktree
	if err := be.Destroy(ctx, backendID); err != nil {
		t.Fatalf("failed to destroy worktree: %v", err)
	}

	// Delete environment from database
	if err := db.DeleteEnvironment(envID); err != nil {
		t.Fatalf("failed to delete environment record: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(backendID); !os.IsNotExist(err) {
		t.Error("worktree was not destroyed")
	}

	// Verify environment is gone from database
	_, err = db.GetEnvironment(envID)
	if err == nil {
		t.Error("expected environment to be deleted from database")
	}
}

// TestEnvListOutput tests the list command output format.
func TestEnvListOutput(t *testing.T) {
	db := openTestDB(t)

	// Create a test environment
	env := &state.Environment{
		ID:         "output1234567890123456789012345",
		Backend:    "local",
		BackendID:  "/path/to/worktree",
		RepoPath:   "/test",
		BranchName: "env/output123456",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusReady,
	}

	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("failed to create environment: %v", err)
	}

	// Get environments
	envs, err := db.ListEnvironments(state.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list environments: %v", err)
	}

	if len(envs) != 1 {
		t.Fatalf("expected 1 environment, got %d", len(envs))
	}

	// Verify environment fields
	gotEnv := envs[0]
	if gotEnv.ID != "output1234567890123456789012345" {
		t.Errorf("expected ID output..., got %s", gotEnv.ID)
	}
	if gotEnv.Status != state.StatusReady {
		t.Errorf("expected status ready, got %s", gotEnv.Status)
	}
	if gotEnv.BranchName != "env/output123456" {
		t.Errorf("expected branch env/output123456, got %s", gotEnv.BranchName)
	}
	if gotEnv.BackendID != "/path/to/worktree" {
		t.Errorf("expected backendID /path/to/worktree, got %s", gotEnv.BackendID)
	}
}

// TestDuplicateEnvironment tests that creating a duplicate environment fails.
func TestDuplicateEnvironment(t *testing.T) {
	db := openTestDB(t)

	env := &state.Environment{
		ID:         "dup123456789012345678901234567",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "env/test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusReady,
	}

	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("first CreateEnvironment failed: %v", err)
	}

	// Try to create again - should fail
	err := db.CreateEnvironment(env)
	if err == nil {
		t.Error("expected error for duplicate environment")
	}
}

// TestBackendRegistration verifies the worktree backend is registered.
func TestBackendRegistration(t *testing.T) {
	be, err := backend.Get(backend.BackendConfig{
		Name: "local",
		Type: "worktree",
	})
	if err != nil {
		t.Fatalf("failed to get worktree backend: %v", err)
	}
	if be == nil {
		t.Error("backend is nil")
	}
}

// TestEnvironmentStatusTransitions tests valid status transitions.
func TestEnvironmentStatusTransitions(t *testing.T) {
	db := openTestDB(t)

	env := &state.Environment{
		ID:         "trans1234567890123456789012345",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "env/test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	// Transition to ready
	env.Status = state.StatusReady
	if err := db.UpdateEnvironment(env); err != nil {
		t.Fatalf("UpdateEnvironment to ready failed: %v", err)
	}

	gotEnv, _ := db.GetEnvironment("trans1234567890123456789012345")
	if gotEnv.Status != state.StatusReady {
		t.Errorf("expected ready, got %s", gotEnv.Status)
	}

	// Transition to removed
	env.Status = state.StatusRemoved
	if err := db.UpdateEnvironment(env); err != nil {
		t.Fatalf("UpdateEnvironment to removed failed: %v", err)
	}

	gotEnv, _ = db.GetEnvironment("trans1234567890123456789012345")
	if gotEnv.Status != state.StatusRemoved {
		t.Errorf("expected removed, got %s", gotEnv.Status)
	}
}

// TestCobraCommands verifies env command is registered with root.
func TestCobraCommands(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if strings.HasPrefix(cmd.Use, "env") {
			found = true
			break
		}
	}
	if !found {
		t.Error("command 'env' not found in root command")
	}
}

// TestEnvListAlias verifies that "ls" is an alias for "list".
func TestEnvListAlias(t *testing.T) {
	// Find the env command
	var envCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if strings.HasPrefix(cmd.Use, "env") {
			envCmd = cmd
			break
		}
	}
	if envCmd == nil {
		t.Fatal("env command not found")
	}

	// Find the list subcommand
	var listCmd *cobra.Command
	for _, cmd := range envCmd.Commands() {
		if strings.HasPrefix(cmd.Use, "list") {
			listCmd = cmd
			break
		}
	}
	if listCmd == nil {
		t.Fatal("list command not found")
	}

	// Verify "ls" is in the aliases
	found := false
	for _, alias := range listCmd.Aliases {
		if alias == "ls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'ls' to be an alias for 'list'")
	}
}
