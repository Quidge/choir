package env

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Quidge/choir/internal/gitutil"
	"github.com/Quidge/choir/internal/state"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List environments",
	Long: `List all environments, optionally filtered by backend or repository.

By default, removed and failed environments are hidden. Use --all to show them.`,
	Args: cobra.NoArgs,
	RunE: runList,
}

var (
	listBackendFlag string
	listRepoFlag    bool
	listAllFlag     bool
)

func init() {
	listCmd.Flags().StringVar(&listBackendFlag, "backend", "", "filter by backend")
	listCmd.Flags().BoolVar(&listRepoFlag, "repo", false, "filter by current repository")
	listCmd.Flags().BoolVar(&listAllFlag, "all", false, "include removed/failed environments")
}

func runList(cmd *cobra.Command, args []string) error {
	// Open state database
	db, err := state.Open("")
	if err != nil {
		return fmt.Errorf("failed to open state database: %w", err)
	}
	defer db.Close()

	// Build list options
	opts := state.ListOptions{
		Backend: listBackendFlag,
	}

	// Filter by current repository if requested
	if listRepoFlag {
		repoRoot, err := gitutil.RepoRoot("")
		if err != nil {
			return fmt.Errorf("not in a git repository: %w", err)
		}
		opts.RepoPath = repoRoot
	}

	// By default, exclude removed and failed environments
	if !listAllFlag {
		opts.Statuses = []state.EnvironmentStatus{
			state.StatusProvisioning,
			state.StatusReady,
		}
	}

	// Get environments
	envs, err := db.ListEnvironments(opts)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(envs) == 0 {
		fmt.Println("No environments found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tBRANCH\tCREATED")
	for _, env := range envs {
		created := formatTimeAgo(env.CreatedAt)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", state.ShortID(env.ID), env.Status, env.BranchName, created)
	}
	w.Flush()

	return nil
}

// formatTimeAgo formats a time as a human-readable relative time.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}
