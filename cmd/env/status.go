package env

import (
	"errors"
	"fmt"

	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status ID",
	Short: "Show detailed environment info",
	Long: `Show detailed information about an environment.

The ID can be a prefix if it uniquely identifies an environment.`,
	Args: cobra.ExactArgs(1),
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
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
		if errors.Is(err, state.ErrInvalidPrefix) {
			return fmt.Errorf("invalid environment ID %q: must contain only hexadecimal characters", idPrefix)
		}
		return fmt.Errorf("failed to get environment: %w", err)
	}

	// Print detailed info
	fmt.Printf("ID:          %s\n", env.ID)
	fmt.Printf("Short ID:    %s\n", state.ShortID(env.ID))
	fmt.Printf("Status:      %s\n", env.Status)
	fmt.Printf("Backend:     %s\n", env.Backend)
	if env.BackendID != "" {
		fmt.Printf("Path:        %s\n", env.BackendID)
	}
	fmt.Printf("Branch:      %s\n", env.BranchName)
	fmt.Printf("Base Branch: %s\n", env.BaseBranch)
	fmt.Printf("Repository:  %s\n", env.RepoPath)
	if env.RemoteURL != "" {
		fmt.Printf("Remote:      %s\n", env.RemoteURL)
	}
	fmt.Printf("Created:     %s\n", env.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}
