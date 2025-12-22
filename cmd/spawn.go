package cmd

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

var spawnCmd = &cobra.Command{
	Use:   "spawn TASK_ID",
	Short: "Create and start a new agent",
	Long: `Create and start a new agent with the given task ID.

The agent runs in an isolated environment with a clone of the current repository
on a dedicated branch (agent/<task-id> by default).`,
	Args: cobra.ExactArgs(1),
	RunE: runSpawn,
}

func init() {
	rootCmd.AddCommand(spawnCmd)

	spawnCmd.Flags().String("base", "", "base branch to spawn from (default: current branch)")
	spawnCmd.Flags().String("prompt", "", "initial prompt to display when agent starts")
	spawnCmd.Flags().String("task-file", "", "file containing task description")
	spawnCmd.Flags().Bool("rm", false, "remove agent automatically when shell exits")
	spawnCmd.Flags().Int("cpus", 0, "override CPU allocation")
	spawnCmd.Flags().String("memory", "", "override memory allocation")
	spawnCmd.Flags().Bool("no-setup", false, "skip setup commands from project config")
}

func runSpawn(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	taskID := args[0]

	// Validate task ID as a valid branch name component
	if err := gitutil.ValidateBranchName(taskID); err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	// Get base branch from flag or current branch
	baseBranch, _ := cmd.Flags().GetString("base")
	noSetup, _ := cmd.Flags().GetBool("no-setup")

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
				return fmt.Errorf("cannot spawn from detached HEAD, use --base to specify a branch")
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
	createCfg, err := config.NewCreateConfig(merged, repoInfo, taskID)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Determine branch name
	branchPrefix := merged.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "agent/"
	}
	branchName := branchPrefix + taskID

	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Check if agent already exists
	existingAgent, err := db.GetAgent(taskID)
	if err == nil && existingAgent != nil {
		return fmt.Errorf("agent %q already exists (status: %s)", taskID, existingAgent.Status)
	}
	if err != nil && !errors.Is(err, state.ErrAgentNotFound) {
		return fmt.Errorf("failed to check for existing agent: %w", err)
	}

	// Create agent record with provisioning status
	agent := &state.Agent{
		TaskID:     taskID,
		Backend:    merged.Backend,
		RepoPath:   repoRoot,
		RemoteURL:  remoteURL,
		BranchName: branchName,
		BaseBranch: baseBranch,
		CreatedAt:  time.Now(),
		Status:     state.StatusProvisioning,
	}

	if err := db.CreateAgent(agent); err != nil {
		return fmt.Errorf("failed to create agent record: %w", err)
	}

	// Get backend
	be, err := backend.Get(backend.BackendConfig{
		Name: merged.Backend,
		Type: merged.BackendType,
	})
	if err != nil {
		// Clean up agent record on failure
		_ = db.DeleteAgent(taskID)
		return fmt.Errorf("failed to get backend: %w", err)
	}

	// Create workspace
	fmt.Printf("Creating worktree for agent %q...\n", taskID)
	backendID, err := be.Create(ctx, &createCfg)
	if err != nil {
		// Mark agent as failed
		agent.Status = state.StatusFailed
		_ = db.UpdateAgent(agent)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Update agent with backendID
	agent.BackendID = backendID
	if err := db.UpdateAgent(agent); err != nil {
		// Try to clean up the worktree
		_ = be.Destroy(ctx, backendID)
		_ = db.DeleteAgent(taskID)
		return fmt.Errorf("failed to update agent record: %w", err)
	}

	// Run setup unless --no-setup is specified
	if !noSetup && len(createCfg.SetupCommands) > 0 {
		fmt.Printf("Running setup commands...\n")
		runner := be.NewSetupRunner(backendID)
		setupCfg := &backend.SetupConfig{
			Environment:   createCfg.Environment,
			Files:         createCfg.Files,
			SetupCommands: createCfg.SetupCommands,
		}
		if err := runner.Run(ctx, setupCfg); err != nil {
			agent.Status = state.StatusFailed
			_ = db.UpdateAgent(agent)
			return fmt.Errorf("setup failed: %w", err)
		}
	}

	// Update agent status to running
	agent.Status = state.StatusRunning
	if err := db.UpdateAgent(agent); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	fmt.Printf("Agent %q created at %s\n", taskID, backendID)
	fmt.Printf("Branch: %s (from %s)\n", branchName, baseBranch)
	fmt.Printf("Dropping into shell...\n\n")

	// Open shell
	if err := be.Shell(ctx, backendID); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}

	return nil
}
