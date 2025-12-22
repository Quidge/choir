package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"absolute path", "/foo/bar", "/foo/bar"},
		{"relative path", "foo/bar", "foo/bar"},
		{"tilde only", "~", home},
		{"tilde with path", "~/foo/bar", home + "/foo/bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExpandEnvVars(t *testing.T) {
	os.Setenv("TEST_VAR", "testvalue")
	os.Setenv("ANOTHER_VAR", "another")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no vars", "plain text", "plain text"},
		{"single var", "${TEST_VAR}", "testvalue"},
		{"var in text", "prefix-${TEST_VAR}-suffix", "prefix-testvalue-suffix"},
		{"multiple vars", "${TEST_VAR}:${ANOTHER_VAR}", "testvalue:another"},
		{"missing var", "${NONEXISTENT}", ""},
		{"default value", "${NONEXISTENT:-default}", "default"},
		{"default with set var", "${TEST_VAR:-default}", "testvalue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReadFromFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "secret")
	if err := os.WriteFile(testFile, []byte("secret-value\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reading the file
	value, err := ReadFromFile(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "secret-value" {
		t.Errorf("expected %q, got %q", "secret-value", value)
	}

	// Test reading non-existent file
	_, err = ReadFromFile(filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoadProjectConfig(t *testing.T) {
	t.Run("missing config returns defaults", func(t *testing.T) {
		cfg, err := LoadProjectConfig("/nonexistent/path/.choir.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Version != 1 {
			t.Errorf("expected version 1, got %d", cfg.Version)
		}
		if cfg.BranchPrefix != "agent/" {
			t.Errorf("expected branch_prefix 'agent/', got %q", cfg.BranchPrefix)
		}
	})

	t.Run("valid config parses correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".choir.yaml")

		content := `version: 1
base_image: ubuntu:22.04
packages:
  - python3
  - nodejs
env:
  NODE_ENV: development
  API_KEY:
    from_file: ~/.secrets/key
files:
  - source: ~/.aws
    target: /home/ubuntu/.aws
    readonly: true
setup:
  - npm install
resources:
  memory: 8GB
  cpus: 8
branch_prefix: feature/
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadProjectConfig(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.BaseImage != "ubuntu:22.04" {
			t.Errorf("expected base_image 'ubuntu:22.04', got %q", cfg.BaseImage)
		}
		if len(cfg.Packages) != 2 {
			t.Errorf("expected 2 packages, got %d", len(cfg.Packages))
		}
		if cfg.Env["NODE_ENV"].Value != "development" {
			t.Errorf("expected NODE_ENV 'development', got %q", cfg.Env["NODE_ENV"].Value)
		}
		if cfg.Env["API_KEY"].FromFile != "~/.secrets/key" {
			t.Errorf("expected API_KEY from_file '~/.secrets/key', got %q", cfg.Env["API_KEY"].FromFile)
		}
		if len(cfg.Files) != 1 {
			t.Errorf("expected 1 file mount, got %d", len(cfg.Files))
		}
		if !cfg.Files[0].ReadOnly {
			t.Error("expected file mount to be readonly")
		}
		if cfg.Resources.Memory != "8GB" {
			t.Errorf("expected memory '8GB', got %q", cfg.Resources.Memory)
		}
		if cfg.BranchPrefix != "feature/" {
			t.Errorf("expected branch_prefix 'feature/', got %q", cfg.BranchPrefix)
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".choir.yaml")

		content := `version: 1
invalid: [yaml: syntax`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := LoadProjectConfig(configPath)
		if err == nil {
			t.Error("expected error for invalid YAML")
		}
	})
}

func TestMerge(t *testing.T) {
	global := GlobalConfig{
		Version:        1,
		DefaultBackend: "local",
		Credentials: CredentialsConfig{
			ClaudeConfig: "~/.claude",
			SSHKeys:      "~/.ssh",
			GitConfig:    "~/.gitconfig",
			GitHubCLI:    "~/.config/gh",
		},
		Backends: map[string]Backend{
			"local": {
				Type:   "lima",
				CPUs:   4,
				Memory: "4GB",
				Disk:   "50GB",
			},
		},
	}

	t.Run("backend defaults", func(t *testing.T) {
		project := DefaultProjectConfig()
		flags := FlagOverrides{}

		merged, err := Merge(global, project, flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if merged.Resources.CPUs != 4 {
			t.Errorf("expected CPUs 4, got %d", merged.Resources.CPUs)
		}
		if merged.Resources.Memory != "4GB" {
			t.Errorf("expected Memory '4GB', got %q", merged.Resources.Memory)
		}
	})

	t.Run("project overrides backend", func(t *testing.T) {
		project := DefaultProjectConfig()
		project.Resources.Memory = "8GB"
		project.Resources.CPUs = 8
		flags := FlagOverrides{}

		merged, err := Merge(global, project, flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if merged.Resources.CPUs != 8 {
			t.Errorf("expected CPUs 8, got %d", merged.Resources.CPUs)
		}
		if merged.Resources.Memory != "8GB" {
			t.Errorf("expected Memory '8GB', got %q", merged.Resources.Memory)
		}
	})

	t.Run("flags override everything", func(t *testing.T) {
		project := DefaultProjectConfig()
		project.Resources.Memory = "8GB"
		flags := FlagOverrides{
			Memory: "16GB",
			CPUs:   16,
		}

		merged, err := Merge(global, project, flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if merged.Resources.CPUs != 16 {
			t.Errorf("expected CPUs 16, got %d", merged.Resources.CPUs)
		}
		if merged.Resources.Memory != "16GB" {
			t.Errorf("expected Memory '16GB', got %q", merged.Resources.Memory)
		}
	})

	t.Run("unknown backend returns error", func(t *testing.T) {
		project := DefaultProjectConfig()
		flags := FlagOverrides{Backend: "nonexistent"}

		_, err := Merge(global, project, flags)
		if err == nil {
			t.Error("expected error for unknown backend")
		}
	})
}

func TestExpandEnvMap(t *testing.T) {
	os.Setenv("TEST_DB_URL", "postgres://localhost/test")
	defer os.Unsetenv("TEST_DB_URL")

	// Create a temp file for from_file test
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "api-key")
	if err := os.WriteFile(secretFile, []byte("secret123\n"), 0644); err != nil {
		t.Fatal(err)
	}

	envVars := map[string]EnvVar{
		"LITERAL":      {Value: "literal-value"},
		"FROM_ENV":     {Value: "${TEST_DB_URL}"},
		"FROM_FILE":    {FromFile: secretFile},
		"WITH_DEFAULT": {Value: "${NONEXISTENT:-fallback}"},
	}

	result, err := ExpandEnvMap(envVars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["LITERAL"] != "literal-value" {
		t.Errorf("expected LITERAL 'literal-value', got %q", result["LITERAL"])
	}
	if result["FROM_ENV"] != "postgres://localhost/test" {
		t.Errorf("expected FROM_ENV 'postgres://localhost/test', got %q", result["FROM_ENV"])
	}
	if result["FROM_FILE"] != "secret123" {
		t.Errorf("expected FROM_FILE 'secret123', got %q", result["FROM_FILE"])
	}
	if result["WITH_DEFAULT"] != "fallback" {
		t.Errorf("expected WITH_DEFAULT 'fallback', got %q", result["WITH_DEFAULT"])
	}
}

func TestDefaultGlobalConfig(t *testing.T) {
	cfg := DefaultGlobalConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.DefaultBackend != "local" {
		t.Errorf("expected default_backend 'local', got %q", cfg.DefaultBackend)
	}
	if cfg.Credentials.ClaudeConfig != "~/.claude" {
		t.Errorf("expected claude_config '~/.claude', got %q", cfg.Credentials.ClaudeConfig)
	}
	if cfg.Backends["local"].Type != "lima" {
		t.Errorf("expected local backend type 'lima', got %q", cfg.Backends["local"].Type)
	}
}

func TestDefaultProjectConfig(t *testing.T) {
	cfg := DefaultProjectConfig()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.BranchPrefix != "agent/" {
		t.Errorf("expected branch_prefix 'agent/', got %q", cfg.BranchPrefix)
	}
}
