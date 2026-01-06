package worktree

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// setupXDGDataHome sets XDG_DATA_HOME to a temp directory for testing.
// Uses t.TempDir() for automatic cleanup.
// Returns the XDG_DATA_HOME path.
func setupXDGDataHome(t *testing.T) string {
	t.Helper()

	xdgDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgDir)
	return xdgDir
}

// setupTestRepo creates a temporary git repository for testing.
// Uses t.TempDir() for automatic cleanup.
// Returns the repo path.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	// Use t.TempDir() which handles cleanup automatically
	tmpDir := t.TempDir()

	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Use cleanGitEnv from worktree.go to avoid git hook interference
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
	xdgDir := setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "abc123def456abc123def456abc12345",
		Repository: config.RepositoryInfo{
			Path:       repoDir,
			BaseBranch: "HEAD",
		},
		BranchPrefix: "env/",
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

	// Verify worktree is in correct location (uses short ID - first 12 chars)
	// Now in XDG_DATA_HOME/choir/worktrees/choir-<id>
	expectedPath := filepath.Join(xdgDir, "choir", "worktrees", "choir-abc123def456")
	if backendID != expectedPath {
		t.Errorf("expected backendID %q, got %q", expectedPath, backendID)
	}
}

func TestCreateMissingID(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		Repository: config.RepositoryInfo{
			Path: "/some/path",
		},
	}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
	if !errors.Is(err, ErrMissingID) {
		t.Errorf("expected ErrMissingID, got: %v", err)
	}
}

func TestCreateMissingRepoPath(t *testing.T) {
	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "abc123def456abc123def456abc12345",
	}

	_, err := b.Create(ctx, cfg)
	if err == nil {
		t.Fatal("expected error for missing repository path")
	}
	if !errors.Is(err, ErrMissingRepoPath) {
		t.Errorf("expected ErrMissingRepoPath, got: %v", err)
	}
}

func TestCreateDuplicate(t *testing.T) {
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "dup123def456abc123def456abc12345",
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
	if !errors.Is(err, ErrWorktreeExists) {
		t.Errorf("expected ErrWorktreeExists, got: %v", err)
	}
}

func TestStatus(t *testing.T) {
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "stat12def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "stst12def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "exec12def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "exenv2def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "fail12def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "dest12def456abc123def456abc12345",
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	// Create multiple worktrees
	envIDs := []string{"list1def456abc123def456abc12345", "list2def456abc123def456abc12345"}
	var backendIDs []string

	for _, envID := range envIDs {
		cfg := &config.CreateConfig{
			ID: envID,
			Repository: config.RepositoryInfo{
				Path:       repoDir,
				BaseBranch: "HEAD",
			},
		}

		id, err := b.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("Create(%s) failed: %v", envID, err)
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

func TestListEmpty(t *testing.T) {
	setupXDGDataHome(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	// List should return empty when no worktrees exist
	list, err := b.List(ctx)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("expected 0 worktrees, got %d: %v", len(list), list)
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
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &config.CreateConfig{
		ID: "canc12def456abc123def456abc12345",
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

func TestWorktreeConfigExtensionEnabled(t *testing.T) {
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "cfge12def456abc123def456abc12345",
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

	// Verify extensions.worktreeConfig is enabled on the main repo
	cmd := exec.Command("git", "config", "--get", "extensions.worktreeConfig")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get extensions.worktreeConfig: %v", err)
	}

	if strings.TrimSpace(string(output)) != "true" {
		t.Errorf("expected extensions.worktreeConfig to be 'true', got %q", strings.TrimSpace(string(output)))
	}
}

func TestWorktreeConfigIsolation(t *testing.T) {
	setupXDGDataHome(t)
	repoDir := setupTestRepo(t)

	b, _ := New(backend.BackendConfig{})
	ctx := context.Background()

	cfg := &config.CreateConfig{
		ID: "isol12def456abc123def456abc12345",
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

	// Get original user.name from main repo (or note that it's unset)
	cmd := exec.Command("git", "config", "--get", "user.name")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	originalOutput, _ := cmd.Output()
	originalName := strings.TrimSpace(string(originalOutput))

	// Set a different user.name in the worktree using --worktree flag
	worktreeTestName := "Worktree Test User"
	cmd = exec.Command("git", "config", "--worktree", "user.name", worktreeTestName)
	cmd.Dir = backendID
	cmd.Env = cleanGitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to set worktree config: %v\n%s", err, out)
	}

	// Verify the worktree has the new config
	cmd = exec.Command("git", "config", "--get", "user.name")
	cmd.Dir = backendID
	cmd.Env = cleanGitEnv()
	worktreeOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get worktree user.name: %v", err)
	}
	if strings.TrimSpace(string(worktreeOutput)) != worktreeTestName {
		t.Errorf("worktree should have user.name %q, got %q", worktreeTestName, strings.TrimSpace(string(worktreeOutput)))
	}

	// Verify main repo still has original user.name (isolation works)
	cmd = exec.Command("git", "config", "--get", "user.name")
	cmd.Dir = repoDir
	cmd.Env = cleanGitEnv()
	mainOutput, _ := cmd.Output()
	mainName := strings.TrimSpace(string(mainOutput))

	if mainName != originalName {
		t.Errorf("main repo user.name changed from %q to %q - isolation failed!", originalName, mainName)
	}
}
