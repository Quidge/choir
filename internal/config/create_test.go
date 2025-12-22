package config

import (
	"testing"
)

func TestValidateFileMounts(t *testing.T) {
	tests := []struct {
		name    string
		files   []FileMount
		wantErr bool
	}{
		{
			name:    "empty list",
			files:   nil,
			wantErr: false,
		},
		{
			name: "valid absolute targets",
			files: []FileMount{
				{Source: "/home/user/.aws", Target: "/home/ubuntu/.aws"},
				{Source: "/tmp/config", Target: "/etc/myapp/config"},
			},
			wantErr: false,
		},
		{
			name: "relative target path",
			files: []FileMount{
				{Source: "/home/user/.aws", Target: "home/ubuntu/.aws"},
			},
			wantErr: true,
		},
		{
			name: "empty target path",
			files: []FileMount{
				{Source: "/home/user/.aws", Target: ""},
			},
			wantErr: true,
		},
		{
			name: "mixed valid and invalid",
			files: []FileMount{
				{Source: "/home/user/.aws", Target: "/home/ubuntu/.aws"},
				{Source: "/tmp/config", Target: "relative/path"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileMounts(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileMounts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCreateConfig(t *testing.T) {
	baseMerged := MergedConfig{
		Backend:     "local",
		BackendType: "lima",
		Credentials: CredentialsConfig{
			ClaudeConfig: "/home/user/.claude",
			SSHKeys:      "/home/user/.ssh",
			GitConfig:    "/home/user/.gitconfig",
			GitHubCLI:    "/home/user/.config/gh",
		},
		Resources: Resources{
			CPUs:   4,
			Memory: "4GB",
			Disk:   "50GB",
		},
		BaseImage:    "ubuntu:22.04",
		Packages:     []string{"python3", "nodejs"},
		Env:          map[string]string{"NODE_ENV": "development"},
		Files:        []FileMount{{Source: "/home/user/.aws", Target: "/home/ubuntu/.aws"}},
		Setup:        []string{"npm install"},
		BranchPrefix: "agent/",
	}

	baseRepo := RepositoryInfo{
		Path:       "/home/user/projects/myapp",
		RemoteURL:  "git@github.com:user/myapp.git",
		BaseBranch: "main",
	}

	t.Run("valid config", func(t *testing.T) {
		cfg, err := NewCreateConfig(baseMerged, baseRepo, "task-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.TaskID != "task-123" {
			t.Errorf("expected TaskID 'task-123', got %q", cfg.TaskID)
		}
		if cfg.Backend != "local" {
			t.Errorf("expected Backend 'local', got %q", cfg.Backend)
		}
		if cfg.BackendType != "lima" {
			t.Errorf("expected BackendType 'lima', got %q", cfg.BackendType)
		}
		if cfg.Repository.Path != "/home/user/projects/myapp" {
			t.Errorf("expected Repository.Path, got %q", cfg.Repository.Path)
		}
		if cfg.Repository.RemoteURL != "git@github.com:user/myapp.git" {
			t.Errorf("expected Repository.RemoteURL, got %q", cfg.Repository.RemoteURL)
		}
		if cfg.Repository.BaseBranch != "main" {
			t.Errorf("expected Repository.BaseBranch 'main', got %q", cfg.Repository.BaseBranch)
		}
		if cfg.Resources.CPUs != 4 {
			t.Errorf("expected CPUs 4, got %d", cfg.Resources.CPUs)
		}
		if len(cfg.Packages) != 2 {
			t.Errorf("expected 2 packages, got %d", len(cfg.Packages))
		}
		if cfg.Environment["NODE_ENV"] != "development" {
			t.Errorf("expected NODE_ENV 'development', got %q", cfg.Environment["NODE_ENV"])
		}
		if len(cfg.SetupCommands) != 1 {
			t.Errorf("expected 1 setup command, got %d", len(cfg.SetupCommands))
		}
	})

	t.Run("empty taskID", func(t *testing.T) {
		_, err := NewCreateConfig(baseMerged, baseRepo, "")
		if err == nil {
			t.Error("expected error for empty taskID")
		}
	})

	t.Run("empty repository path", func(t *testing.T) {
		emptyRepo := RepositoryInfo{
			RemoteURL:  "git@github.com:user/myapp.git",
			BaseBranch: "main",
		}
		_, err := NewCreateConfig(baseMerged, emptyRepo, "task-123")
		if err == nil {
			t.Error("expected error for empty repository path")
		}
	})

	t.Run("invalid file mount target", func(t *testing.T) {
		invalidMerged := baseMerged
		invalidMerged.Files = []FileMount{
			{Source: "/home/user/.aws", Target: "relative/path"},
		}
		_, err := NewCreateConfig(invalidMerged, baseRepo, "task-123")
		if err == nil {
			t.Error("expected error for invalid file mount target")
		}
	})

	t.Run("empty remote URL is allowed", func(t *testing.T) {
		repoNoRemote := RepositoryInfo{
			Path:       "/home/user/projects/myapp",
			BaseBranch: "main",
		}
		cfg, err := NewCreateConfig(baseMerged, repoNoRemote, "task-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Repository.RemoteURL != "" {
			t.Errorf("expected empty RemoteURL, got %q", cfg.Repository.RemoteURL)
		}
	})

	t.Run("nil environment map", func(t *testing.T) {
		noEnvMerged := baseMerged
		noEnvMerged.Env = nil
		cfg, err := NewCreateConfig(noEnvMerged, baseRepo, "task-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Environment != nil {
			t.Errorf("expected nil Environment, got %v", cfg.Environment)
		}
	})

	t.Run("empty environment map", func(t *testing.T) {
		emptyEnvMerged := baseMerged
		emptyEnvMerged.Env = map[string]string{}
		cfg, err := NewCreateConfig(emptyEnvMerged, baseRepo, "task-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Environment == nil {
			t.Error("expected non-nil Environment for empty map")
		}
		if len(cfg.Environment) != 0 {
			t.Errorf("expected empty Environment, got %v", cfg.Environment)
		}
	})
}
