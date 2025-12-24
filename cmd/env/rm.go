package env

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm ID",
	Short: "Remove an environment",
	Long: `Remove an environment and destroy its worktree.

The ID can be a prefix if it uniquely identifies an environment.
This removes the worktree directory and deletes the environment from the database.

For ready environments, confirmation is required unless -f is used.`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

var rmForceFlag bool

func init() {
	rmCmd.Flags().BoolVarP(&rmForceFlag, "force", "f", false, "skip confirmation for ready environments")
}

func runRm(cmd *cobra.Command, args []string) error {
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

	shortID := state.ShortID(env.ID)

	// Confirm for ready environments unless -f is used
	if env.Status == state.StatusReady && !rmForceFlag {
		fmt.Printf("Environment %s is ready. Remove it? [y/N] ", shortID)
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// If environment has a backendID, destroy the worktree
	if env.BackendID != "" {
		// Get backend - for MVP, always use worktree
		be, err := backend.Get(backend.BackendConfig{
			Name: env.Backend,
			Type: "worktree",
		})
		if err != nil {
			return fmt.Errorf("failed to get backend: %w", err)
		}

		if err := be.Destroy(ctx, env.BackendID); err != nil {
			// Log the error but continue to delete the environment record
			fmt.Fprintf(os.Stderr, "warning: failed to destroy worktree: %v\n", err)
		}
	}

	// Delete environment from database
	if err := db.DeleteEnvironment(env.ID); err != nil {
		return fmt.Errorf("failed to delete environment record: %w", err)
	}

	fmt.Printf("Removed %s\n", shortID)
	return nil
}
