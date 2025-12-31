//go:build conformance

package conformance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Quidge/choir/internal/backend"
	"github.com/Quidge/choir/internal/config"
)

// ConformanceSuite defines all conformance tests for any Backend implementation.
// It verifies that a backend correctly implements the Backend interface contract.
type ConformanceSuite struct {
	// Backend under test.
	Backend backend.Backend

	// RepoSetup is called to create a git repo for each test.
	// Should use t.Cleanup() for automatic cleanup.
	RepoSetup func(t *testing.T) string
}

// Run executes all conformance tests.
func (s *ConformanceSuite) Run(t *testing.T) {
	t.Run("Lifecycle", s.testLifecycle)
	t.Run("FileMounts", s.testFileMounts)
	t.Run("Environment", s.testEnvironment)
	t.Run("SetupCommands", s.testSetupCommands)
}

// testLifecycle tests basic backend lifecycle operations.
func (s *ConformanceSuite) testLifecycle(t *testing.T) {
	t.Run("CreateAndDestroy", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		// Verify it exists and is running
		status, err := s.Backend.Status(env.Ctx, env.BackendID)
		if err != nil {
			t.Fatalf("Status() returned error: %v", err)
		}
		if status.State != backend.StateRunning {
			t.Errorf("expected state Running, got %v", status.State)
		}

		// Verify Exec works
		output, exitCode, err := s.Backend.Exec(env.Ctx, env.BackendID, "echo hello")
		if err != nil {
			t.Fatalf("Exec() returned error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("expected exit code 0, got %d", exitCode)
		}
		if !strings.Contains(output, "hello") {
			t.Errorf("expected output to contain 'hello', got: %s", output)
		}
	})

	t.Run("StatusNotFound", func(t *testing.T) {
		status, err := s.Backend.Status(t.Context(), "/nonexistent/conformance-test-path")
		if err != nil {
			t.Fatalf("Status() should not error for missing workspace: %v", err)
		}
		if status.State != backend.StateNotFound {
			t.Errorf("expected StateNotFound, got %v", status.State)
		}
	})

	t.Run("ExecOnNonexistent", func(t *testing.T) {
		_, _, err := s.Backend.Exec(t.Context(), "/nonexistent/conformance-test-path", "echo test")
		if err == nil {
			t.Error("expected error for exec on nonexistent workspace")
		}
	})
}

// testFileMounts tests file mounting behavior.
// THIS IS THE CRITICAL TEST SUITE - it would have caught the relative path bug.
func (s *ConformanceSuite) testFileMounts(t *testing.T) {
	t.Run("RelativeTargetPath", func(t *testing.T) {
		// THIS TEST WOULD HAVE CAUGHT THE BUG in issue #46
		// Relative target paths should work - the backend handles them
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["simple"], Target: "config/app.txt", ReadOnly: true},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Verify file exists at relative path within workspace
		env.AssertFileExists("config/app.txt")
		env.AssertFileContent("config/app.txt", "hello world")
	})

	t.Run("AbsoluteTargetPath", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		// Use an absolute path inside the workspace
		absTarget := fmt.Sprintf("%s/absolute-test.txt", env.BackendID)

		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["simple"], Target: absTarget, ReadOnly: true},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		env.AssertFileExists(absTarget)
		env.AssertFileContent(absTarget, "hello world")
	})

	t.Run("ReadOnlyMount", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["simple"], Target: "readonly.txt", ReadOnly: true},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// For worktree backend, readonly creates symlinks
		env.AssertSymlink("readonly.txt")
		env.AssertFileContent("readonly.txt", "hello world")
	})

	t.Run("WritableMount", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["simple"], Target: "writable.txt", ReadOnly: false},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Writable mount should be a copy, not symlink
		env.AssertNotSymlink("writable.txt")
		env.AssertFileContent("writable.txt", "hello world")

		// Should be writable
		env.MustExec("echo ' modified' >> writable.txt")
		output := env.MustExec("cat writable.txt")
		if !strings.Contains(output, "modified") {
			t.Error("file should be writable")
		}
	})

	t.Run("DirectoryMount", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["config-dir"], Target: "imported-config", ReadOnly: false},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		env.AssertDirectory("imported-config")
		env.AssertFileExists("imported-config/app.yaml")
		env.AssertFileContent("imported-config/app.yaml", "key: value")
		env.AssertFileExists("imported-config/nested/deep.txt")
		env.AssertFileContent("imported-config/nested/deep.txt", "deep content")
	})

	t.Run("NestedTargetPath", func(t *testing.T) {
		// Target in non-existent directory should create parent dirs
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		fixtures := CreateTestFixtures(t, t.TempDir())
		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: fixtures["simple"], Target: "deep/nested/path/file.txt", ReadOnly: true},
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		env.AssertFileExists("deep/nested/path/file.txt")
		env.AssertFileContent("deep/nested/path/file.txt", "hello world")
	})

	t.Run("SourceNotFound", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Files: []config.FileMount{
				{Source: "/nonexistent/source/file.txt", Target: "dest.txt", ReadOnly: true},
			},
		})
		if err == nil {
			t.Error("expected error for missing source file")
		}
	})
}

// testEnvironment tests environment variable handling.
func (s *ConformanceSuite) testEnvironment(t *testing.T) {
	t.Run("BasicEnvVar", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{
				"MY_VAR": "my_value",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		env.AssertEnvVar("MY_VAR", "my_value")
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{
				"QUOTED": "it's got 'quotes'",
				"DOLLAR": "$NOT_EXPANDED",
				"SPACES": "value with spaces",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		env.AssertEnvVar("QUOTED", "it's got 'quotes'")
		env.AssertEnvVar("DOLLAR", "$NOT_EXPANDED")
		env.AssertEnvVar("SPACES", "value with spaces")
	})

	t.Run("EnvVarPersistence", func(t *testing.T) {
		// Env vars should persist across Exec calls
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{
				"PERSISTENT": "value",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Multiple exec calls should all see the var
		for i := 0; i < 3; i++ {
			env.AssertEnvVar("PERSISTENT", "value")
		}
	})

	t.Run("EmptyValue", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{
				"EMPTY": "",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// Variable should be set but empty
		output := env.MustExec("[ -z \"${EMPTY+x}\" ] && echo UNSET || echo SET")
		if !strings.Contains(output, "SET") {
			t.Error("empty env var should be set")
		}
		env.AssertEnvVar("EMPTY", "")
	})

	t.Run("EmptyEnvironment", func(t *testing.T) {
		// No environment variables should not create .choir-env file
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		// .choir-env should not exist
		env.AssertFileNotExists(".choir-env")
	})
}

// testSetupCommands tests setup command execution.
func (s *ConformanceSuite) testSetupCommands(t *testing.T) {
	t.Run("ExecutionOrder", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			SetupCommands: []string{
				"echo 'first' > order.log",
				"echo 'second' >> order.log",
				"echo 'third' >> order.log",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		output := env.MustExec("cat order.log")
		expected := "first\nsecond\nthird"
		if strings.TrimSpace(output) != expected {
			t.Errorf("commands ran out of order: got %q, want %q", strings.TrimSpace(output), expected)
		}
	})

	t.Run("WorkingDirectory", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			SetupCommands: []string{
				"pwd > pwd.log",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		output := env.MustExec("cat pwd.log")
		if strings.TrimSpace(output) != env.BackendID {
			t.Errorf("working directory wrong: got %q, want %q", strings.TrimSpace(output), env.BackendID)
		}
	})

	t.Run("EnvVarsAvailable", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			Environment: map[string]string{
				"SETUP_VAR": "available",
			},
			SetupCommands: []string{
				"echo $SETUP_VAR > var.log",
			},
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		output := env.MustExec("cat var.log")
		if strings.TrimSpace(output) != "available" {
			t.Errorf("env var not available in setup: got %q", strings.TrimSpace(output))
		}
	})

	t.Run("FailureStopsExecution", func(t *testing.T) {
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			SetupCommands: []string{
				"echo 'before' > fail.log",
				"exit 1",
				"echo 'after' >> fail.log",
			},
		})
		if err == nil {
			t.Fatal("expected error for failing command")
		}

		output := env.MustExec("cat fail.log")
		if strings.Contains(output, "after") {
			t.Error("commands after failure should not run")
		}
	})

	t.Run("EmptyCommands", func(t *testing.T) {
		// No setup commands should succeed
		repoPath := s.RepoSetup(t)
		env := NewTestEnv(t, s.Backend, repoPath)

		err := env.RunSetup(&backend.SetupConfig{
			SetupCommands: []string{},
		})
		if err != nil {
			t.Fatalf("empty commands should succeed: %v", err)
		}
	})
}
