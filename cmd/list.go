package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Quidge/choir/internal/gitutil"
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	Long: `List all agents, optionally filtered by backend or repository.

By default, removed and failed agents are hidden. Use --all to show them.`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().String("backend", "", "filter by backend")
	listCmd.Flags().Bool("repo", false, "filter by current repository")
	listCmd.Flags().Bool("all", false, "include removed/failed agents")
	listCmd.Flags().Bool("json", false, "output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	filterBackend, _ := cmd.Flags().GetString("backend")
	filterRepo, _ := cmd.Flags().GetBool("repo")
	showAll, _ := cmd.Flags().GetBool("all")

	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Build list options
	opts := state.ListOptions{
		Backend: filterBackend,
	}

	// Filter by current repository if requested
	if filterRepo {
		repoRoot, err := gitutil.RepoRoot("")
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}
		opts.RepoPath = repoRoot
	}

	// By default, exclude removed and failed agents
	if !showAll {
		opts.Statuses = []state.Status{
			state.StatusProvisioning,
			state.StatusRunning,
			state.StatusStopped,
		}
	}

	// Get agents
	agents, err := db.ListAgents(opts)
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents) == 0 {
		fmt.Println("No agents found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TASK ID\tSTATUS\tBRANCH\tPATH")
	for _, agent := range agents {
		path := agent.BackendID
		if path == "" {
			path = "(pending)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", agent.TaskID, agent.Status, agent.BranchName, path)
	}
	w.Flush()

	return nil
}
