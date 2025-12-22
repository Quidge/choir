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

var attachCmd = &cobra.Command{
	Use:   "attach TASK_ID",
	Short: "Connect to a running agent",
	Long: `Connect to an existing agent's shell.

When you exit the shell, the agent continues running.`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
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

	// Check agent status
	switch agent.Status {
	case state.StatusRemoved:
		return fmt.Errorf("agent %q has been removed", taskID)
	case state.StatusFailed:
		return fmt.Errorf("agent %q is in failed state", taskID)
	case state.StatusProvisioning:
		return fmt.Errorf("agent %q is still provisioning", taskID)
	}

	if agent.BackendID == "" {
		return fmt.Errorf("agent %q has no backend ID (may not be fully provisioned)", taskID)
	}

	// Get backend - for MVP, always use worktree
	be, err := backend.Get(backend.BackendConfig{
		Name: agent.Backend,
		Type: "worktree",
	})
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	fmt.Printf("Attaching to agent %q at %s...\n\n", taskID, agent.BackendID)

	// Open shell
	if err := be.Shell(ctx, agent.BackendID); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
