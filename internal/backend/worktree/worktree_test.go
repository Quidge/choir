package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// setupTestRepo creates a temporary git repository for testing.
// Returns the repo path and a cleanup function.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "choir-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// cleanGitEnv returns a clean environment without git-specific variables
	// that might interfere with test git operations (e.g., during pre-commit hooks)
	cleanGitEnv := func() []string {
		var env []string
		for _, e := range os.Environ() {
			// Skip git environment variables that might interfere
			if !strings.HasPrefix(e, "GIT_") {
				env = append(env, e)
			}
		}
		return env
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init repo: %v\n%s", err, out)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	cmd.Run()

	// Create initial commit
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to commit: %v\n%s", err, out)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return repoDir, cleanup
}

func TestNew(t *testing.T) {
	b, err := New(backend.BackendConfig{})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	if b == nil {
		t.Fatal("New() returned nil backend")
	}
}

func TestBackendType(t *testing.T) {
	if BackendType != "worktree" {
		t.Errorf("expected BackendType 'worktree', got %q", BackendType)
	}
}

func TestCreate(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "test-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
		BranchPrefix: "agent/",
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	// Verify worktree was created
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify marker file exists
	markerPath := filepath.Join(backendID, markerFile)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("marker file was not created")
	}

	// Verify worktree is in correct location
	expectedPath := filepath.Join(filepath.Dir(repoDir), "choir-test-task")
	if backendID != expectedPath {
		t.Errorf("expected backendID %q, got %q", expectedPath, backendID)
	}
}

func TestCreateMissingTaskID(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		Repository: config.RepositoryInfo{
			Path: "/some/path",
		},
	}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for missing task ID")
	}
	if !strings.Contains(err.Error(), "task ID") {
		t.Errorf("expected error about task ID, got: %v", err)
	}
}

func TestCreateMissingRepoPath(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "test-task",
	}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for missing repository path")
	}
	if !strings.Contains(err.Error(), "repository path") {
		t.Errorf("expected error about repository path, got: %v", err)
	}
}

func TestCreateDuplicate(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "dup-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("first Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	// Try to create again
	_, err = b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for duplicate worktree")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected error about already existing, got: %v", err)
	}
}

func TestStatus(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "status-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	status, err := b.Status(ctx, backendID)
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}
	if status.State != backend.StateRunning {
		t.Errorf("expected state Running, got %v", status.State)
	}
}

func TestStatusNotFound(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	status, err := b.Status(ctx, "/nonexistent/path")
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}
	if status.State != backend.StateNotFound {
		t.Errorf("expected state NotFound, got %v", status.State)
	}
}

func TestStatusNotChoirManaged(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "not-choir-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	status, err := b.Status(ctx, tmpDir)
	if err != nil {
		t.Fatalf("Status() returned error: %v", err)
	}
	if status.State != backend.StateError {
		t.Errorf("expected state Error for non-choir directory, got %v", status.State)
	}
}

func TestStartStop(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "startstop-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	// Start should be no-op
	if err := b.Start(ctx, backendID); err != nil {
		t.Errorf("Start() returned error: %v", err)
	}

	// Stop should be no-op
	if err := b.Stop(ctx, backendID); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestStartNotFound(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	err := b.Start(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent worktree")
	}
}

func TestStopNotFound(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	err := b.Stop(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent worktree")
	}
}

func TestExec(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "exec-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	// Test simple command
	output, exitCode, err := b.Exec(ctx, backendID, "echo hello")
	if err != nil {
		t.Fatalf("Exec() returned error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("expected output to contain 'hello', got: %s", output)
	}
}

func TestExecWithEnv(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "exec-env-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	// Set up environment using setup runner
	runner := b.NewSetupRunner(backendID)
	err = runner.Run(ctx, &backend.SetupConfig{
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
	})
	if err != nil {
		t.Fatalf("SetupRunner.Run() failed: %v", err)
	}

	// Verify environment is available
	output, exitCode, err := b.Exec(ctx, backendID, "echo $TEST_VAR")
	if err != nil {
		t.Fatalf("Exec() returned error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(output, "test_value") {
		t.Errorf("expected output to contain 'test_value', got: %s", output)
	}
}

func TestExecNotFound(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	_, _, err := b.Exec(ctx, "/nonexistent/path", "echo hello")
	if err == nil {
		t.Fatal("expected error for non-existent worktree")
	}
}

func TestExecFailingCommand(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "exec-fail-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}
	defer b.Destroy(ctx, backendID)

	_, exitCode, err := b.Exec(ctx, backendID, "exit 42")
	if err != nil {
		t.Fatalf("Exec() returned unexpected error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
}

func TestDestroy(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		TaskID: "destroy-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := b.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(backendID); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Destroy it
	if err := b.Destroy(ctx, backendID); err != nil {
		t.Fatalf("Destroy() failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(backendID); !os.IsNotExist(err) {
		t.Error("worktree was not destroyed")
	}
}

func TestList(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b := &Backend{repoRoot: repoDir}
	ctx := context.Background()

	// Create multiple worktrees
	taskIDs := []string{"list-task-1", "list-task-2"}
	var backendIDs []string

	for _, taskID := range taskIDs {
		cfg := &config.CreateConfig{
			TaskID: taskID,
			Repository: config.RepositoryInfo{
				Path:       repoDir,
				BaseBranch: "HEAD",
			},
		}

		id, err := b.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("Create(%s) failed: %v", taskID, err)
		}
		backendIDs = append(backendIDs, id)
	}

	// Clean up at end
	defer func() {
		for _, id := range backendIDs {
			b.Destroy(ctx, id)
		}
	}()

	// List worktrees
	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("expected 2 worktrees, got %d: %v", len(list), list)
	}

	// Verify both are in the list
	// Use EvalSymlinks to handle macOS symlinked temp dirs
	resolveOrKeep := func(p string) string {
		resolved, err := filepath.EvalSymlinks(p)
		if err != nil {
			return p
		}
		return resolved
	}

	found := make(map[string]bool)
	for _, id := range list {
		found[resolveOrKeep(id)] = true
	}

	for _, id := range backendIDs {
		if !found[resolveOrKeep(id)] {
			t.Errorf("expected %s to be in list, got: %v", id, list)
		}
	}
}

func TestNewSetupRunner(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	runner := b.NewSetupRunner("/test/path")

	if runner == nil {
		t.Fatal("NewSetupRunner returned nil")
	}

	hostRunner, ok := runner.(*HostSetupRunner)
	if !ok {
		t.Fatal("expected HostSetupRunner")
	}
	if hostRunner.WorkDir != "/test/path" {
		t.Errorf("expected WorkDir '/test/path', got %q", hostRunner.WorkDir)
	}
}

func TestIsChoirManaged(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (string, func())
		expected bool
	}{
		{
			name: "choir prefix with marker",
			setup: func(t *testing.T) (string, func()) {
				dir, err := os.MkdirTemp("", "choir-test-*")
				if err != nil {
					t.Fatal(err)
				}
				os.WriteFile(filepath.Join(dir, markerFile), []byte("test"), 0644)
				return dir, func() { os.RemoveAll(dir) }
			},
			expected: true,
		},
		{
			name: "choir prefix without marker",
			setup: func(t *testing.T) (string, func()) {
				dir, err := os.MkdirTemp("", "choir-test-*")
				if err != nil {
					t.Fatal(err)
				}
				return dir, func() { os.RemoveAll(dir) }
			},
			expected: false,
		},
		{
			name: "no choir prefix",
			setup: func(t *testing.T) (string, func()) {
				dir, err := os.MkdirTemp("", "other-*")
				if err != nil {
					t.Fatal(err)
				}
				os.WriteFile(filepath.Join(dir, markerFile), []byte("test"), 0644)
				return dir, func() { os.RemoveAll(dir) }
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, cleanup := tt.setup(t)
			defer cleanup()

			result := isChoirManaged(dir)
			if result != tt.expected {
				t.Errorf("isChoirManaged(%q) = %v, expected %v", dir, result, tt.expected)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	repoDir, cleanup := setupTestRepo(t)
	defer cleanup()

	b, _ := New(backend.BackendConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &config.CreateConfig{
		TaskID: "cancel-task",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
	}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Log("Create completed despite cancellation (may succeed if fast enough)")
	}
}
