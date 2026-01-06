//go:build conformance && worktree

package conformance

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
)

// TestWorktreeConformance runs the conformance test suite against the worktree backend,
// followed by worktree-specific tests.
//
// Run with: go test -tags=conformance,worktree ./internal/backend/conformance
func TestWorktreeConformance(t *testing.T) {
	// Set up XDG_DATA_HOME to a temp directory to avoid polluting user's config
	SetupXDGDataHome(t)

	be, err := backend.Get(backend.BackendConfig{
		Name: "conformance-test",
		Type: "worktree",
	})
	if err != nil {
		t.Fatalf("failed to get worktree backend: %v", err)
	}

	suite := &ConformanceSuite{
		Backend:     be,
		BackendType: "worktree",
		RepoSetup:   SetupGitRepo,
	}

	// Run generic Backend interface conformance tests
	suite.Run(t)

	// Run worktree-specific tests (not part of generic Backend interface)
	t.Run("WorktreeSpecific", func(t *testing.T) {
		testConfigIsolation(t, be)
	})
}

// testConfigIsolation verifies that the worktree backend enables
// extensions.worktreeConfig, allowing per-worktree git configuration that
// doesn't pollute the main repository's .git/config.
func testConfigIsolation(t *testing.T, be backend.Backend) {
	repoPath := SetupGitRepo(t)
	env := NewTestEnv(t, be, repoPath, TestEnvConfig{BackendType: "worktree"})

	t.Run("ExtensionEnabled", func(t *testing.T) {
		// Verify extensions.worktreeConfig is enabled on the main repo
		cmd := exec.Command("git", "config", "--get", "extensions.worktreeConfig")
		cmd.Dir = repoPath
		cmd.Env = cleanGitEnv()
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("extensions.worktreeConfig not set on main repo: %v", err)
		}
		if strings.TrimSpace(string(output)) != "true" {
			t.Errorf("expected extensions.worktreeConfig=true, got %q", strings.TrimSpace(string(output)))
		}
	})

	t.Run("ConfigIsolation", func(t *testing.T) {
		// Get original user.name from main repo
		cmd := exec.Command("git", "config", "--get", "user.name")
		cmd.Dir = repoPath
		cmd.Env = cleanGitEnv()
		originalOutput, _ := cmd.Output()
		originalName := strings.TrimSpace(string(originalOutput))

		// Set a different user.name in the worktree using --worktree flag
		testName := "Conformance Test Agent"
		cmd = exec.Command("git", "config", "--worktree", "user.name", testName)
		cmd.Dir = env.BackendID
		cmd.Env = cleanGitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to set worktree config: %v\n%s", err, out)
		}

		// Verify worktree has the new config
		cmd = exec.Command("git", "config", "--get", "user.name")
		cmd.Dir = env.BackendID
		cmd.Env = cleanGitEnv()
		worktreeOutput, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get worktree user.name: %v", err)
		}
		if strings.TrimSpace(string(worktreeOutput)) != testName {
			t.Errorf("worktree user.name: got %q, want %q", strings.TrimSpace(string(worktreeOutput)), testName)
		}

		// Verify main repo is unchanged (isolation works)
		cmd = exec.Command("git", "config", "--get", "user.name")
		cmd.Dir = repoPath
		cmd.Env = cleanGitEnv()
		mainOutput, _ := cmd.Output()
		mainName := strings.TrimSpace(string(mainOutput))

		if mainName != originalName {
			t.Errorf("main repo user.name changed from %q to %q - config isolation failed", originalName, mainName)
		}
	})
}
