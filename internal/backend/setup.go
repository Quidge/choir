package backend

import (
	"context"

	"github.com/Quidge/choir/internal/config"
)

// SetupRunner abstracts workspace setup steps. Each backend provides its own
// implementation that knows how to execute in that environment.
//
// Different backends execute setup differently:
//
//	| Backend  | How Setup Runs                           |
//	|----------|------------------------------------------|
//	| Worktree | Direct shell commands on host            |
//	| Lima     | cloud-init (boot) + SSH (post-boot)      |
//	| EC2      | user-data + SSM commands                 |
//
// By having each backend provide its own SetupRunner, the complexity of
// *how* to execute stays encapsulated. The caller just calls runner.Run(config).
type SetupRunner interface {
	// Run executes all setup steps for the workspace.
	Run(ctx context.Context, cfg *SetupConfig) error
}

// SetupConfig contains the configuration for setting up a workspace.
type SetupConfig struct {
	// Environment contains environment variables to set in the workspace.
	Environment map[string]string

	// Files contains files to copy or link into the workspace.
	Files []config.FileMount

	// SetupCommands contains commands to run after environment setup.
	SetupCommands []string
}
