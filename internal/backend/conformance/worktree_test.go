//go:build conformance && worktree

package conformance

import (
	"testing"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
)

// TestWorktreeConformance runs the conformance test suite against the worktree backend.
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
	suite.Run(t)
}
