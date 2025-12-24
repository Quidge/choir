package config

import (
	"gopkg.in/yaml.v3"
)

// GlobalConfig represents the global configuration loaded from
// ~/.config/choir/config.yaml
type GlobalConfig struct {
	Version        int                `yaml:"version"`
	DefaultBackend string             `yaml:"default_backend"`
	Credentials    CredentialsConfig  `yaml:"credentials"`
	Backends       map[string]Backend `yaml:"backends"`
}

// CredentialsConfig defines paths to credential files/directories.
type CredentialsConfig struct {
	ClaudeConfig string `yaml:"claude_config"`
	SSHKeys      string `yaml:"ssh_keys"`
	GitConfig    string `yaml:"git_config"`
	GitHubCLI    string `yaml:"github_cli"`
}

// Backend represents configuration for a VM backend.
type Backend struct {
	Type   string `yaml:"type"`
	CPUs   int    `yaml:"cpus"`
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`
	VMType string `yaml:"vm_type"` // Lima-specific: vz or qemu
}

// ProjectConfig represents the project configuration loaded from
// .choir.yaml in the repository root.
type ProjectConfig struct {
	Version      int               `yaml:"version"`
	BaseImage    string            `yaml:"base_image"`
	Packages     []string          `yaml:"packages"`
	Env          map[string]EnvVar `yaml:"env"`
	Files        []FileMount       `yaml:"files"`
	Setup        []string          `yaml:"setup"`
	Resources    Resources         `yaml:"resources"`
	BranchPrefix string            `yaml:"branch_prefix"`
}

// EnvVar represents an environment variable value.
// It can be either a literal string or a from_file reference.
type EnvVar struct {
	Value    string // Literal value (after expansion)
	FromFile string // Path to file containing value
}

// UnmarshalYAML implements custom unmarshaling for EnvVar to handle
// both string values and {from_file: path} objects.
func (e *EnvVar) UnmarshalYAML(value *yaml.Node) error {
	// Try unmarshaling as a simple string first
	var str string
	if err := value.Decode(&str); err == nil {
		e.Value = str
		return nil
	}

	// Try unmarshaling as an object with from_file
	var obj struct {
		FromFile string `yaml:"from_file"`
	}
	if err := value.Decode(&obj); err != nil {
		return err
	}
	e.FromFile = obj.FromFile
	return nil
}

// FileMount represents a file or directory to copy into the VM.
type FileMount struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	ReadOnly bool   `yaml:"readonly"`
}

// Resources represents resource allocation overrides.
type Resources struct {
	Memory string `yaml:"memory"`
	CPUs   int    `yaml:"cpus"`
	Disk   string `yaml:"disk"`
}

// MergedConfig represents the final merged configuration
// after applying precedence rules (backend defaults → global → project → flags).
type MergedConfig struct {
	// Backend selection
	Backend     string
	BackendType string

	// Credentials (from global config)
	Credentials CredentialsConfig

	// Resources (merged from all sources)
	Resources Resources

	// Project-specific settings
	BaseImage    string
	Packages     []string
	Env          map[string]string // Expanded environment variables
	Files        []FileMount
	Setup        []string
	BranchPrefix string
}

// RepositoryInfo contains information about the git repository.
type RepositoryInfo struct {
	// Path is the absolute path to the repository root.
	Path string

	// RemoteURL is the URL of the origin remote.
	// May be empty if no remote is configured.
	RemoteURL string

	// BaseBranch is the branch to base new work on.
	BaseBranch string
}

// CreateConfig is the unified configuration passed to Backend.Create().
// It combines data from global config, project config, CLI flags, and git repository info.
//
// Not all fields apply to all backends. The struct is complete, but backends
// may ignore fields that don't apply to them. See the backend applicability table:
//
//	| Field            | Worktree         | Lima             |
//	|------------------|------------------|------------------|
//	| ID               | ✓ Used           | ✓ Used           |
//	| Resources.*      | Ignored (no VM)  | ✓ Used           |
//	| Credentials.*    | Ignored (host)   | ✓ Used           |
//	| Repository.*     | ✓ Used           | ✓ Used           |
//	| Environment      | ✓ Used (export)  | ✓ Used           |
//	| Files            | ✓ Used (symlink) | ✓ Used           |
//	| Packages         | Warn if present  | ✓ Used           |
//	| SetupCommands    | ✓ Used (on host) | ✓ Used           |
type CreateConfig struct {
	// ID is the unique identifier for this environment (32 hex chars).
	ID string

	// Backend is the name of the backend to use (e.g., "local").
	Backend string

	// BackendType is the type of backend (e.g., "lima", "worktree").
	BackendType string

	// Resources contains resource allocation settings.
	// Worktree backend ignores these (no VM).
	Resources Resources

	// Credentials contains paths to credential files/directories.
	// Worktree backend ignores these (uses host credentials).
	Credentials CredentialsConfig

	// Repository contains git repository information.
	Repository RepositoryInfo

	// BaseImage is the VM base image (e.g., "ubuntu:22.04").
	// Only used by Lima backend.
	BaseImage string

	// Packages are system packages to install.
	// Worktree backend warns if present.
	Packages []string

	// Environment contains expanded environment variables to set.
	Environment map[string]string

	// Files are file/directory mounts to copy into the environment.
	Files []FileMount

	// SetupCommands are commands to run after environment setup.
	SetupCommands []string

	// BranchPrefix is the prefix for environment branch names (default: "env/").
	BranchPrefix string
}

// DefaultGlobalConfig returns a GlobalConfig with sensible defaults.
func DefaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
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
				VMType: "vz",
			},
		},
	}
}

// DefaultProjectConfig returns a ProjectConfig with sensible defaults.
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Version:      1,
		BranchPrefix: "env/",
	}
}
