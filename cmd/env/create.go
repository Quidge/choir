package env

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Quidge/choir/internal/backend"
	_ "github.com/Quidge/choir/internal/backend/worktree" // Register worktree backend
	"github.com/Quidge/choir/internal/config"
	"github.com/Quidge/choir/internal/gitutil"
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new environment",
	Long: `Create a new environment with a unique ID.

The environment runs in an isolated workspace with a clone of the current repository
on a dedicated branch (env/<short-id> by default).

The environment ID is printed on success for scripting use.`,
	Args: cobra.NoArgs,
	RunE: runCreate,
}

var (
	baseFlag    string
	backendFlag string
	noSetupFlag bool
)

func init() {
	createCmd.Flags().StringVar(&baseFlag, "base", "", "base branch to create from (default: current branch)")
	createCmd.Flags().StringVar(&backendFlag, "backend", "", "override default backend")
	createCmd.Flags().BoolVar(&noSetupFlag, "no-setup", false, "skip setup commands from project config")
}

func runCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Generate environment ID
	envID, err := state.GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate environment ID: %w", err)
	}
	shortID := state.ShortID(envID)

	// Get base branch from flag or current branch
	baseBranch := baseFlag

	// Get repository info
	repoRoot, err := gitutil.RepoRoot("")
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	remoteURL, _ := gitutil.RemoteURL(repoRoot, "origin")

	if baseBranch == "" {
		baseBranch, err = gitutil.CurrentBranch(repoRoot)
		if err != nil {
			if errors.Is(err, gitutil.ErrDetachedHead) {
				return fmt.Errorf("cannot create environment from detached HEAD, use --base to specify a branch")
			}
			return fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	// Load configuration
	merged, err := config.LoadFromCwd(config.FlagOverrides{
		Backend: backendFlag,
	})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// For MVP, force worktree backend
	merged.BackendType = "worktree"

	// Build repository info
	repoInfo := config.RepositoryInfo{
		Path:       repoRoot,
		RemoteURL:  remoteURL,
		BaseBranch: baseBranch,
	}

	// Build CreateConfig
	createCfg, err := config.NewCreateConfig(merged, repoInfo, envID)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Determine branch name
	branchPrefix := merged.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "env/"
	}
	branchName := branchPrefix + shortID

	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Create environment record with provisioning status
	env := &state.Environment{
		ID:         envID,
		Backend:    merged.Backend,
		RepoPath:   repoRoot,
		RemoteURL:  remoteURL,
		BranchName: branchName,
		BaseBranch: baseBranch,
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateEnvironment(env); err != nil {
		return fmt.Errorf("failed to create environment record: %w", err)
	}

	// Get backend
	be, err := backend.Get(backend.BackendConfig{
		Name: merged.Backend,
		Type: merged.BackendType,
	})
	if err != nil {
		// Clean up environment record on failure
		_ = db.DeleteEnvironment(envID)
		return fmt.Errorf("failed to get backend: %w", err)
	}

	// Create workspace
	backendID, err := be.Create(ctx, &createCfg)
	if err != nil {
		// Mark environment as failed
		env.Status = state.StatusFailed
		_ = db.UpdateEnvironment(env)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Update environment with backendID
	env.BackendID = backendID
	if err := db.UpdateEnvironment(env); err != nil {
		// Try to clean up the worktree
		_ = be.Destroy(ctx, backendID)
		_ = db.DeleteEnvironment(envID)
		return fmt.Errorf("failed to update environment record: %w", err)
	}

	// Run setup unless --no-setup is specified
	// Setup handles environment variables, file mounts, and setup commands
	hasSetupWork := len(createCfg.SetupCommands) > 0 ||
		len(createCfg.Files) > 0 ||
		len(createCfg.Environment) > 0
	if !noSetupFlag && hasSetupWork {
		runner := be.NewSetupRunner(backendID)
		setupCfg := &backend.SetupConfig{
			Environment:   createCfg.Environment,
			Files:         createCfg.Files,
			SetupCommands: createCfg.SetupCommands,
		}
		if err := runner.Run(ctx, setupCfg); err != nil {
			env.Status = state.StatusFailed
			_ = db.UpdateEnvironment(env)
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Update environment status to ready
	env.Status = state.StatusReady
	if err := db.UpdateEnvironment(env); err != nil {
		return fmt.Errorf("failed to update environment status: %w", err)
	}

	// Print just the short ID for scripting
	fmt.Println(shortID)

	return nil
}
