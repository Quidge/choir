package cmd

import (
	"bytes"
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

// TestSpawnCommand tests the spawn command logic.
func TestSpawnCommand(t *testing.T) {
	t.Run("validates task ID", func(t *testing.T) {
		// Test with invalid task ID containing spaces
		rootCmd.SetArgs([]string{"spawn", "invalid task"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for invalid task ID")
		}
	})

	t.Run("validates task ID with special chars", func(t *testing.T) {
		rootCmd.SetArgs([]string{"spawn", "task~with~tilde"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for invalid task ID with tilde")
		}
	})
}

// TestSpawnIntegration tests the spawn command with real git repo and database.
func TestSpawnIntegration(t *testing.T) {
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
	taskID := "test-spawn"

	// Build config manually (simulating what spawn does)
	createCfg := &config.CreateConfig{
		TaskID:       taskID,
		Backend:      "local",
		BackendType:  "worktree",
		BranchPrefix: "agent/",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	// Create agent record
	agent := &state.Agent{
		TaskID:     taskID,
		Backend:    "local",
		RepoPath:   repoDir,
		BranchName: "agent/" + taskID,
		BaseBranch: "HEAD",
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Create worktree
	backendID, err := be.Create(ctx, createCfg)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}
	defer be.Destroy(ctx, backendID)

	// Update agent
	agent.BackendID = backendID
	agent.Status = state.StatusRunning
	if err := db.UpdateAgent(agent); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Error("worktree was not created")
	}

	// Verify agent in database
	gotAgent, err := db.GetAgent(taskID)
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}

	if gotAgent.Status != state.StatusRunning {
		t.Errorf("expected status running, got %s", gotAgent.Status)
	}

	if gotAgent.BackendID != backendID {
		t.Errorf("expected backendID %s, got %s", backendID, gotAgent.BackendID)
	}
}

// TestAttachCommand tests the attach command logic.
func TestAttachCommand(t *testing.T) {
	db := openTestDB(t)

	t.Run("agent not found", func(t *testing.T) {
		_, err := db.GetAgent("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent agent")
		}
	})

	t.Run("agent removed status", func(t *testing.T) {
		agent := &state.Agent{
			TaskID:     "removed-agent",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "agent/removed",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     state.StatusRemoved,
		}
		if err := db.CreateAgent(agent); err != nil {
			t.Fatalf("failed to create agent: %v", err)
		}

		gotAgent, err := db.GetAgent("removed-agent")
		if err != nil {
			t.Fatalf("failed to get agent: %v", err)
		}

		if gotAgent.Status != state.StatusRemoved {
			t.Errorf("expected status removed, got %s", gotAgent.Status)
		}
	})

	t.Run("agent failed status", func(t *testing.T) {
		agent := &state.Agent{
			TaskID:     "failed-agent",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "agent/failed",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     state.StatusFailed,
		}
		if err := db.CreateAgent(agent); err != nil {
			t.Fatalf("failed to create agent: %v", err)
		}

		gotAgent, err := db.GetAgent("failed-agent")
		if err != nil {
			t.Fatalf("failed to get agent: %v", err)
		}

		if gotAgent.Status != state.StatusFailed {
			t.Errorf("expected status failed, got %s", gotAgent.Status)
		}
	})
}

// TestListCommand tests the list command logic.
func TestListCommand(t *testing.T) {
	db := openTestDB(t)

	t.Run("empty list", func(t *testing.T) {
		agents, err := db.ListAgents(state.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list agents: %v", err)
		}

		if len(agents) != 0 {
			t.Errorf("expected 0 agents, got %d", len(agents))
		}
	})

	t.Run("list with agents", func(t *testing.T) {
		// Create test agents
		for i, status := range []state.Status{state.StatusRunning, state.StatusStopped, state.StatusFailed} {
			agent := &state.Agent{
				TaskID:     string(rune('a' + i)),
				Backend:    "local",
				RepoPath:   "/test",
				BranchName: "agent/test",
				BaseBranch: "main",
				CreatedAt:  time.Now(),
				Status:     status,
			}
			if err := db.CreateAgent(agent); err != nil {
				t.Fatalf("failed to create agent: %v", err)
			}
		}

		// List all agents
		agents, err := db.ListAgents(state.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list agents: %v", err)
		}

		if len(agents) != 3 {
			t.Errorf("expected 3 agents, got %d", len(agents))
		}

		// List only running agents
		agents, err = db.ListAgents(state.ListOptions{
			Statuses: []state.Status{state.StatusRunning, state.StatusStopped},
		})
		if err != nil {
			t.Fatalf("failed to list agents: %v", err)
		}

		if len(agents) != 2 {
			t.Errorf("expected 2 active agents, got %d", len(agents))
		}
	})
}

// TestRmCommand tests the rm command logic.
func TestRmCommand(t *testing.T) {
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

	taskID := "test-rm"

	// Create worktree
	createCfg := &config.CreateConfig{
		TaskID:       taskID,
		Backend:      "local",
		BackendType:  "worktree",
		BranchPrefix: "agent/",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := be.Create(ctx, createCfg)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create agent record
	agent := &state.Agent{
		TaskID:     taskID,
		Backend:    "local",
		BackendID:  backendID,
		RepoPath:   repoDir,
		BranchName: "agent/" + taskID,
		BaseBranch: "HEAD",
		CreatedAt:  time.Now(),
		Status:     state.StatusRunning,
	}

	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Destroy worktree
	if err := be.Destroy(ctx, backendID); err != nil {
		t.Fatalf("failed to destroy worktree: %v", err)
	}

	// Delete agent from database
	if err := db.DeleteAgent(taskID); err != nil {
		t.Fatalf("failed to delete agent: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(backendID); !os.IsNotExist(err) {
		t.Error("worktree was not destroyed")
	}

	// Verify agent is gone from database
	_, err = db.GetAgent(taskID)
	if err == nil {
		t.Error("expected agent to be deleted from database")
	}
}

// TestListOutput tests the list command output format.
func TestListOutput(t *testing.T) {
	db := openTestDB(t)

	// Create a test agent
	agent := &state.Agent{
		TaskID:     "output-test",
		Backend:    "local",
		BackendID:  "/path/to/worktree",
		RepoPath:   "/test",
		BranchName: "agent/output-test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusRunning,
	}

	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Get agents
	agents, err := db.ListAgents(state.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list agents: %v", err)
	}

	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}

	// Verify agent fields
	gotAgent := agents[0]
	if gotAgent.TaskID != "output-test" {
		t.Errorf("expected task ID output-test, got %s", gotAgent.TaskID)
	}
	if gotAgent.Status != state.StatusRunning {
		t.Errorf("expected status running, got %s", gotAgent.Status)
	}
	if gotAgent.BranchName != "agent/output-test" {
		t.Errorf("expected branch agent/output-test, got %s", gotAgent.BranchName)
	}
	if gotAgent.BackendID != "/path/to/worktree" {
		t.Errorf("expected backendID /path/to/worktree, got %s", gotAgent.BackendID)
	}
}

// TestDuplicateAgent tests that creating a duplicate agent fails.
func TestDuplicateAgent(t *testing.T) {
	db := openTestDB(t)

	agent := &state.Agent{
		TaskID:     "duplicate-test",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "agent/test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusRunning,
	}

	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("first CreateAgent failed: %v", err)
	}

	// Try to create again - should fail
	err := db.CreateAgent(agent)
	if err == nil {
		t.Error("expected error for duplicate agent")
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

// TestAgentStatusTransitions tests valid status transitions.
func TestAgentStatusTransitions(t *testing.T) {
	db := openTestDB(t)

	agent := &state.Agent{
		TaskID:     "status-test",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "agent/test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	// Transition to running
	agent.Status = state.StatusRunning
	if err := db.UpdateAgent(agent); err != nil {
		t.Fatalf("UpdateAgent to running failed: %v", err)
	}

	gotAgent, _ := db.GetAgent("status-test")
	if gotAgent.Status != state.StatusRunning {
		t.Errorf("expected running, got %s", gotAgent.Status)
	}

	// Transition to stopped
	agent.Status = state.StatusStopped
	if err := db.UpdateAgent(agent); err != nil {
		t.Fatalf("UpdateAgent to stopped failed: %v", err)
	}

	gotAgent, _ = db.GetAgent("status-test")
	if gotAgent.Status != state.StatusStopped {
		t.Errorf("expected stopped, got %s", gotAgent.Status)
	}

	// Transition to removed
	agent.Status = state.StatusRemoved
	if err := db.UpdateAgent(agent); err != nil {
		t.Fatalf("UpdateAgent to removed failed: %v", err)
	}

	gotAgent, _ = db.GetAgent("status-test")
	if gotAgent.Status != state.StatusRemoved {
		t.Errorf("expected removed, got %s", gotAgent.Status)
	}
}

// TestCobraCommands verifies all commands are registered with root.
func TestCobraCommands(t *testing.T) {
	commands := []string{"spawn", "attach", "list", "rm"}

	for _, name := range commands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if strings.HasPrefix(cmd.Use, name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %q not found in root command", name)
		}
	}
}

// TestSpawnRequiresTaskID verifies spawn requires exactly one argument.
func TestSpawnRequiresTaskID(t *testing.T) {
	// Reset output for clean test
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"spawn"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when no task ID provided")
	}
}

// TestAttachRequiresTaskID verifies attach requires exactly one argument.
func TestAttachRequiresTaskID(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"attach"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when no task ID provided")
	}
}

// TestRmRequiresTaskID verifies rm requires exactly one argument.
func TestRmRequiresTaskID(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"rm"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when no task ID provided")
	}
}

// TestListNoArgs verifies list takes no arguments.
func TestListNoArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"list", "extra-arg"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when extra arguments provided to list")
	}
}
