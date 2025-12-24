package env

import (
	"context"
	"errors"
	"fmt"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach ID",
	Short: "Enter an existing environment",
	Long: `Enter an existing environment's shell.

The ID can be a prefix if it uniquely identifies an environment.
When you exit the shell, the environment continues to exist.`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func runAttach(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	idPrefix := args[0]

	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Get environment from database by prefix
	env, err := db.GetEnvironmentByPrefix(idPrefix)
	if err != nil {
		if errors.Is(err, state.ErrEnvironmentNotFound) {
			return fmt.Errorf("environment %q not found", idPrefix)
		}
		if errors.Is(err, state.ErrAmbiguousPrefix) {
			return fmt.Errorf("ambiguous environment ID %q: matches multiple environments", idPrefix)
		}
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Check environment status
	switch env.Status {
	case state.StatusRemoved:
		return fmt.Errorf("environment %q has been removed", state.ShortID(env.ID))
	case state.StatusFailed:
		return fmt.Errorf("environment %q is in failed state", state.ShortID(env.ID))
	case state.StatusProvisioning:
		return fmt.Errorf("environment %q is still provisioning", state.ShortID(env.ID))
	}

	if env.BackendID == "" {
		return fmt.Errorf("environment %q has no backend ID (may not be fully provisioned)", state.ShortID(env.ID))
	}

	// Get backend - for MVP, always use worktree
	be, err := backend.Get(backend.BackendConfig{
		Name: env.Backend,
		Type: "worktree",
	})
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	// Open shell
	if err := be.Shell(ctx, env.BackendID); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
