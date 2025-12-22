package gitutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing.
// Returns the path to the repo root and a cleanup function.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit (required for branch operations)
	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

func TestRepoRoot(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	repoDirResolved, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	t.Run("from repo root", func(t *testing.T) {
		root, err := RepoRoot(repoDir)
		if err != nil {
			t.Fatalf("RepoRoot() failed: %v", err)
		}
		if root != repoDirResolved {
			t.Errorf("RepoRoot() = %q, want %q", root, repoDirResolved)
		}
	})

	t.Run("from subdirectory", func(t *testing.T) {
		subDir := filepath.Join(repoDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		root, err := RepoRoot(subDir)
		if err != nil {
			t.Fatalf("RepoRoot() from subdir failed: %v", err)
		}
		if root != repoDirResolved {
			t.Errorf("RepoRoot() = %q, want %q", root, repoDirResolved)
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		notGitDir := t.TempDir()
		_, err := RepoRoot(notGitDir)
		if !errors.Is(err, ErrNotGitRepo) {
			t.Errorf("RepoRoot() error = %v, want ErrNotGitRepo", err)
		}
	})

	t.Run("empty dir uses current working directory", func(t *testing.T) {
		// This test uses the actual current directory (which is the choir repo)
		root, err := RepoRoot("")
		if err != nil {
			t.Fatalf("RepoRoot(\"\") failed: %v", err)
		}
		if root == "" {
			t.Error("RepoRoot(\"\") returned empty string")
		}
	})
}

func TestCurrentBranch(t *testing.T) {
	repoDir := setupTestRepo(t)

	t.Run("default branch", func(t *testing.T) {
		branch, err := CurrentBranch(repoDir)
		if err != nil {
			t.Fatalf("CurrentBranch() failed: %v", err)
		}
		// Default branch could be 'main' or 'master' depending on git config
		if branch != "main" && branch != "master" {
			t.Errorf("CurrentBranch() = %q, want main or master", branch)
		}
	})

	t.Run("feature branch", func(t *testing.T) {
		cmd := exec.Command("git", "checkout", "-b", "feature/test-branch")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git checkout -b failed: %v", err)
		}

		branch, err := CurrentBranch(repoDir)
		if err != nil {
			t.Fatalf("CurrentBranch() failed: %v", err)
		}
		if branch != "feature/test-branch" {
			t.Errorf("CurrentBranch() = %q, want feature/test-branch", branch)
		}
	})

	t.Run("detached HEAD", func(t *testing.T) {
		// Get the current commit hash
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = repoDir
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git rev-parse failed: %v", err)
		}

		// Checkout the commit directly (detached HEAD)
		cmd = exec.Command("git", "checkout", string(out[:40]))
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git checkout commit failed: %v", err)
		}

		_, err = CurrentBranch(repoDir)
		if !errors.Is(err, ErrDetachedHead) {
			t.Errorf("CurrentBranch() error = %v, want ErrDetachedHead", err)
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		notGitDir := t.TempDir()
		_, err := CurrentBranch(notGitDir)
		if err == nil {
			t.Error("CurrentBranch() expected error for non-git directory")
		}
	})
}

func TestIsDetachedHead(t *testing.T) {
	repoDir := setupTestRepo(t)

	t.Run("not detached", func(t *testing.T) {
		if IsDetachedHead(repoDir) {
			t.Error("IsDetachedHead() = true, want false for normal branch")
		}
	})

	t.Run("detached HEAD", func(t *testing.T) {
		// Get the current commit hash
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = repoDir
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git rev-parse failed: %v", err)
		}

		// Checkout the commit directly (detached HEAD)
		cmd = exec.Command("git", "checkout", string(out[:40]))
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git checkout commit failed: %v", err)
		}

		if !IsDetachedHead(repoDir) {
			t.Error("IsDetachedHead() = false, want true for detached HEAD")
		}
	})
}

func TestRemoteURL(t *testing.T) {
	repoDir := setupTestRepo(t)

	t.Run("no remote", func(t *testing.T) {
		_, err := RemoteURL(repoDir, "origin")
		if !errors.Is(err, ErrNoRemote) {
			t.Errorf("RemoteURL() error = %v, want ErrNoRemote", err)
		}
	})

	t.Run("with remote", func(t *testing.T) {
		cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git remote add failed: %v", err)
		}

		url, err := RemoteURL(repoDir, "origin")
		if err != nil {
			t.Fatalf("RemoteURL() failed: %v", err)
		}
		if url != "https://github.com/test/repo.git" {
			t.Errorf("RemoteURL() = %q, want https://github.com/test/repo.git", url)
		}
	})

	t.Run("default remote name", func(t *testing.T) {
		url, err := RemoteURL(repoDir, "")
		if err != nil {
			t.Fatalf("RemoteURL() with empty name failed: %v", err)
		}
		if url != "https://github.com/test/repo.git" {
			t.Errorf("RemoteURL() = %q, want https://github.com/test/repo.git", url)
		}
	})
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple name", "main", true},
		{"with hyphen", "feature-branch", true},
		{"with underscore", "feature_branch", true},
		{"with slash", "feature/branch", true},
		{"with dots", "v1.0.0", true},
		{"alphanumeric", "fix123", true},
		{"complex", "feature/issue-123_fix", true},

		{"empty", "", false},
		{"with space", "feature branch", false},
		{"with tilde", "feature~branch", false},
		{"with caret", "feature^branch", false},
		{"with colon", "feature:branch", false},
		{"with question", "feature?branch", false},
		{"with asterisk", "feature*branch", false},
		{"with bracket", "feature[branch", false},
		{"with backslash", "feature\\branch", false},
		{"double dots", "feature..branch", false},
		{"double slash", "feature//branch", false},
		{"starts with slash", "/feature", false},
		{"ends with slash", "feature/", false},
		{"starts with dot", ".feature", false},
		{"ends with dot", "feature.", false},
		{"ends with .lock", "feature.lock", false},
		{"with @{", "feature@{branch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidBranchName(tt.input)
			if got != tt.want {
				t.Errorf("IsValidBranchName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		err := ValidateBranchName("feature/test-123")
		if err != nil {
			t.Errorf("ValidateBranchName() error = %v, want nil", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		err := ValidateBranchName("")
		if err == nil {
			t.Error("ValidateBranchName(\"\") expected error")
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		err := ValidateBranchName("feature branch")
		if err == nil {
			t.Error("ValidateBranchName(\"feature branch\") expected error")
		}
	})
}

func TestIsInsideWorkTree(t *testing.T) {
	repoDir := setupTestRepo(t)

	t.Run("inside work tree", func(t *testing.T) {
		if !IsInsideWorkTree(repoDir) {
			t.Error("IsInsideWorkTree() = false, want true")
		}
	})

	t.Run("subdirectory", func(t *testing.T) {
		subDir := filepath.Join(repoDir, "src")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		if !IsInsideWorkTree(subDir) {
			t.Error("IsInsideWorkTree() = false for subdir, want true")
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		notGitDir := t.TempDir()
		if IsInsideWorkTree(notGitDir) {
			t.Error("IsInsideWorkTree() = true for non-git dir, want false")
		}
	})
}
