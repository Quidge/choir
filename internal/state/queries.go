package state

import (
	"fmt"
	"strings"
)

// ListOptions specifies filters for listing agents.
type ListOptions struct {
	RepoPath string   // Filter by repository path (exact match)
	Backend  string   // Filter by backend name
	Statuses []Status // Filter by status (any of these)
}

// ListAgents returns all agents matching the given filters.
// If no filters are specified, returns all agents.
func (db *DB) ListAgents(opts ListOptions) ([]*Agent, error) {
	query := `
		SELECT task_id, backend, backend_id, repo_path, remote_url,
		       branch_name, base_branch, created_at, status, prompt, notes
		FROM agents
	`

	var conditions []string
	var args []any

	if opts.RepoPath != "" {
		conditions = append(conditions, "repo_path = ?")
		args = append(args, opts.RepoPath)
	}

	if opts.Backend != "" {
		conditions = append(conditions, "backend = ?")
		args = append(args, opts.Backend)
	}

	if len(opts.Statuses) > 0 {
		// Build parameterized IN clause. This is safe from SQL injection because
		// we generate one "?" placeholder per status and pass values via args.
		placeholders := make([]string, len(opts.Statuses))
		for i, s := range opts.Statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return agents, nil
}

// CountAgents returns the number of agents matching the given filters.
func (db *DB) CountAgents(opts ListOptions) (int, error) {
	query := "SELECT COUNT(*) FROM agents"

	var conditions []string
	var args []any

	if opts.RepoPath != "" {
		conditions = append(conditions, "repo_path = ?")
		args = append(args, opts.RepoPath)
	}

	if opts.Backend != "" {
		conditions = append(conditions, "backend = ?")
		args = append(args, opts.Backend)
	}

	if len(opts.Statuses) > 0 {
		// Build parameterized IN clause. This is safe from SQL injection because
		// we generate one "?" placeholder per status and pass values via args.
		placeholders := make([]string, len(opts.Statuses))
		for i, s := range opts.Statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count agents: %w", err)
	}

	return count, nil
}
