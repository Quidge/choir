package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm TASK_ID",
	Short: "Remove an agent",
	Long: `Remove an agent and destroy its worktree.

This removes the worktree directory and deletes the agent from the database.`,
	Args: cobra.ExactArgs(1),
	RunE: runRm,
}

func init() {
	rootCmd.AddCommand(rmCmd)

	rmCmd.Flags().BoolP("force", "f", false, "remove without confirmation")
}

func runRm(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Get agent from database
	agent, err := db.GetAgent(taskID)
	if err != nil {
		if errors.Is(err, state.ErrAgentNotFound) {
			return fmt.Errorf("agent %q not found", taskID)
		}
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// If agent has a backendID, destroy the worktree
	if agent.BackendID != "" {
		// Get backend - for MVP, always use worktree
		be, err := backend.Get(backend.BackendConfig{
			Name: agent.Backend,
			Type: "worktree",
		})
		if err != nil {
			return fmt.Errorf("failed to get backend: %w", err)
		}

		fmt.Printf("Removing worktree at %s...\n", agent.BackendID)
		if err := be.Destroy(ctx, agent.BackendID); err != nil {
			// Log the error but continue to delete the agent record
			fmt.Printf("Warning: failed to destroy worktree: %v\n", err)
		}
	}

	// Delete agent from database
	if err := db.DeleteAgent(taskID); err != nil {
		return fmt.Errorf("failed to delete agent record: %w", err)
	}

	fmt.Printf("Agent %q removed.\n", taskID)
	return nil
}
