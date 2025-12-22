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
		BranchPrefix: "agent/",
	}
}
