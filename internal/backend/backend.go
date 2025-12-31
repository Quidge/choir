// Package backend defines the interfaces that all choir backends must implement.
// This abstraction allows choir to support multiple backends (worktree, Lima,
// future EC2, etc.) with a uniform interface.
package backend

import (
	"context"

	"github.com/Quidge/choir/internal/config"
)

// Backend defines the interface that all choir backends must implement.
// All backends must implement all methods. If a method doesn't make sense
// for a backend, it should be a no-op rather than returning an error.
//
// Method implementations by backend:
//
//	| Method          | Worktree              | Lima              |
//	|-----------------|-----------------------|-------------------|
//	| Create          | git worktree add      | Provision VM      |
//	| NewSetupRunner  | Returns HostSetupRunner | Returns LimaSetupRunner |
//	| Start           | No-op (always ready)  | Start VM          |
//	| Stop            | No-op                 | Stop VM           |
//	| Destroy         | git worktree remove   | Destroy VM        |
//	| Shell           | cd <dir> && $SHELL    | SSH into VM       |
//	| Exec            | Run in directory      | SSH + run         |
//	| Status          | Check dir exists      | Query VM state    |
//	| List            | git worktree list     | List VMs          |
type Backend interface {
	// Create provisions a new workspace (worktree, VM, etc.)
	Create(ctx context.Context, cfg *config.CreateConfig) (backendID string, err error)

	// NewSetupRunner returns a runner for executing setup steps in this workspace.
	NewSetupRunner(backendID string) SetupRunner

	// Start starts a stopped workspace.
	Start(ctx context.Context, backendID string) error

	// Stop stops a running workspace.
	Stop(ctx context.Context, backendID string) error

	// Destroy permanently destroys a workspace.
	Destroy(ctx context.Context, backendID string) error

	// Shell opens an interactive shell (blocks until exit).
	Shell(ctx context.Context, backendID string) error

	// Exec runs a command and returns output.
	Exec(ctx context.Context, backendID string, command string) (output string, exitCode int, err error)

	// Status queries workspace status.
	Status(ctx context.Context, backendID string) (BackendStatus, error)

	// List returns all choir-managed workspaces.
	List(ctx context.Context) ([]string, error)
}

// BackendStatus represents the current state of a backend workspace.
type BackendStatus struct {
	// State is the current state of the workspace.
	State WorkspaceState

	// Message provides additional context about the current state.
	Message string
}

// WorkspaceState represents the possible states of a workspace.
type WorkspaceState string

const (
	// StateRunning indicates the workspace is running and ready.
	StateRunning WorkspaceState = "running"

	// StateStopped indicates the workspace is stopped but can be started.
	StateStopped WorkspaceState = "stopped"

	// StateCreating indicates the workspace is being created.
	StateCreating WorkspaceState = "creating"

	// StateStopping indicates the workspace is in the process of stopping.
	StateStopping WorkspaceState = "stopping"

	// StateStarting indicates the workspace is in the process of starting.
	StateStarting WorkspaceState = "starting"

	// StateDestroying indicates the workspace is being destroyed.
	StateDestroying WorkspaceState = "destroying"

	// StateNotFound indicates the workspace does not exist.
	StateNotFound WorkspaceState = "not_found"

	// StateError indicates the workspace is in an error state.
	StateError WorkspaceState = "error"
)
