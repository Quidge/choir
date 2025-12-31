//go:build conformance

// Package conformance provides backend-agnostic conformance tests that verify
// backends correctly implement the Backend interface contract.
package conformance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// DefaultTimeout is the default timeout for test operations.
const DefaultTimeout = 30 * time.Second

// TestEnvConfig configures a test environment.
type TestEnvConfig struct {
	// BackendType is the type of backend (e.g., "worktree", "lima").
	BackendType string

	// Timeout for test operations. Defaults to DefaultTimeout.
	Timeout time.Duration
}

// TestEnv encapsulates a complete test environment with assertion helpers.
// It provides a convenient API for conformance tests to verify backend behavior.
type TestEnv struct {
	T         *testing.T
	Backend   backend.Backend
	BackendID string
	RepoPath  string
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// NewTestEnv creates a fully provisioned test environment.
// The environment is automatically cleaned up when the test completes.
func NewTestEnv(t *testing.T, be backend.Backend, repoPath string, cfg TestEnvConfig) *TestEnv {
	t.Helper()

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// Use t.Context() for proper test cancellation propagation (Go 1.21+)
	ctx, cancel := context.WithTimeout(t.Context(), timeout)

	envID := generateTestID(t)

	createCfg := &config.CreateConfig{
		ID:           envID,
		Backend:      "test",
		BackendType:  cfg.BackendType,
		BranchPrefix: "test/",
		Repository: config.RepositoryInfo{
			Path:       repoPath,
			BaseBranch: "HEAD",
		},
	}

	backendID, err := be.Create(ctx, createCfg)
	if err != nil {
		cancel()
		t.Fatalf("failed to create backend: %v", err)
	}

	t.Cleanup(func() {
		// Use a fresh context for cleanup since the test context may be cancelled
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		be.Destroy(cleanupCtx, backendID)
		cancel()
	})

	return &TestEnv{
		T:         t,
		Backend:   be,
		BackendID: backendID,
		RepoPath:  repoPath,
		Ctx:       ctx,
		Cancel:    cancel,
	}
}

// RunSetup executes setup with the given config.
func (e *TestEnv) RunSetup(cfg *backend.SetupConfig) error {
	runner := e.Backend.NewSetupRunner(e.BackendID)
	return runner.Run(e.Ctx, cfg)
}

// Exec runs a command and returns output, exit code, and any error.
func (e *TestEnv) Exec(command string) (string, int, error) {
	return e.Backend.Exec(e.Ctx, e.BackendID, command)
}

// MustExec runs a command and fails the test if it errors or returns non-zero.
func (e *TestEnv) MustExec(command string) string {
	e.T.Helper()
	output, exitCode, err := e.Exec(command)
	if err != nil {
		e.T.Fatalf("command %q failed: %v", command, err)
	}
	if exitCode != 0 {
		e.T.Fatalf("command %q exited with %d: %s", command, exitCode, output)
	}
	return output
}

// AssertFileExists fails if the file doesn't exist in the workspace.
func (e *TestEnv) AssertFileExists(path string) {
	e.T.Helper()
	output, exitCode, _ := e.Exec(fmt.Sprintf("test -e %q && echo OK", path))
	if exitCode != 0 || !strings.Contains(output, "OK") {
		e.T.Errorf("file %q does not exist", path)
	}
}

// AssertFileNotExists fails if the file exists in the workspace.
func (e *TestEnv) AssertFileNotExists(path string) {
	e.T.Helper()
	_, exitCode, _ := e.Exec(fmt.Sprintf("test -e %q", path))
	if exitCode == 0 {
		e.T.Errorf("file %q should not exist", path)
	}
}

// AssertFileContent fails if file content doesn't match expected.
func (e *TestEnv) AssertFileContent(path, expected string) {
	e.T.Helper()
	output := e.MustExec(fmt.Sprintf("cat %q", path))
	if strings.TrimSpace(output) != expected {
		e.T.Errorf("file %q: got %q, want %q", path, strings.TrimSpace(output), expected)
	}
}

// AssertEnvVar fails if environment variable doesn't match expected value.
func (e *TestEnv) AssertEnvVar(name, expected string) {
	e.T.Helper()
	// Use printf to avoid newline issues
	output := e.MustExec(fmt.Sprintf("printf '%%s' \"$%s\"", name))
	if output != expected {
		e.T.Errorf("env var %s: got %q, want %q", name, output, expected)
	}
}

// AssertSymlink fails if path is not a symlink.
func (e *TestEnv) AssertSymlink(path string) {
	e.T.Helper()
	_, exitCode, _ := e.Exec(fmt.Sprintf("test -L %q", path))
	if exitCode != 0 {
		e.T.Errorf("%q is not a symlink", path)
	}
}

// AssertNotSymlink fails if path is a symlink.
func (e *TestEnv) AssertNotSymlink(path string) {
	e.T.Helper()
	_, exitCode, _ := e.Exec(fmt.Sprintf("test -L %q", path))
	if exitCode == 0 {
		e.T.Errorf("%q should not be a symlink", path)
	}
}

// AssertDirectory fails if path is not a directory.
func (e *TestEnv) AssertDirectory(path string) {
	e.T.Helper()
	_, exitCode, _ := e.Exec(fmt.Sprintf("test -d %q", path))
	if exitCode != 0 {
		e.T.Errorf("%q is not a directory", path)
	}
}

// SetupGitRepo creates a temporary git repository for testing.
// Uses t.Cleanup() for automatic cleanup.
// Returns the absolute path to the repo.
func SetupGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	env := cleanGitEnv()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	// Create initial file and commit
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "Initial commit")

	return repoDir
}

// CreateTestFixtures creates standard test fixtures in the given directory.
// Returns a map of fixture name to absolute path.
func CreateTestFixtures(t *testing.T, dir string) map[string]string {
	t.Helper()
	fixtures := make(map[string]string)

	// Simple text file
	simplePath := filepath.Join(dir, "simple.txt")
	if err := os.WriteFile(simplePath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create fixture: %v", err)
	}
	fixtures["simple"] = simplePath

	// Directory with nested files
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(filepath.Join(configDir, "nested"), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "app.yaml"), []byte("key: value"), 0644); err != nil {
		t.Fatalf("failed to create app.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "nested", "deep.txt"), []byte("deep content"), 0644); err != nil {
		t.Fatalf("failed to create deep.txt: %v", err)
	}
	fixtures["config-dir"] = configDir

	// File with special characters (for escaping tests)
	specialPath := filepath.Join(dir, "special.txt")
	if err := os.WriteFile(specialPath, []byte("it's got \"quotes\" and $vars"), 0644); err != nil {
		t.Fatalf("failed to create special.txt: %v", err)
	}
	fixtures["special"] = specialPath

	return fixtures
}

// cleanGitEnv returns environment variables with GIT_* variables removed
// to avoid interference from the test environment.
func cleanGitEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GIT_") {
			env = append(env, e)
		}
	}
	return env
}

// generateTestID generates a 32-character hex ID for testing.
func generateTestID(t *testing.T) string {
	t.Helper()
	// Use a deterministic but unique ID based on test name
	// This avoids needing crypto/rand in tests
	h := fmt.Sprintf("%x", time.Now().UnixNano())
	// Pad to 32 characters
	for len(h) < 32 {
		h = "0" + h
	}
	if len(h) > 32 {
		h = h[:32]
	}
	return h
}
