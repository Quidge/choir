package state

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EnvironmentStatus represents the state of an environment.
type EnvironmentStatus string

const (
	StatusProvisioning EnvironmentStatus = "provisioning"
	StatusReady        EnvironmentStatus = "ready"
	StatusFailed       EnvironmentStatus = "failed"
	StatusRemoved      EnvironmentStatus = "removed"
)

// ValidStatuses contains all valid environment status values.
var ValidStatuses = []EnvironmentStatus{
	StatusProvisioning,
	StatusReady,
	StatusFailed,
	StatusRemoved,
}

// IsValidStatus returns true if s is a valid status.
func IsValidStatus(s EnvironmentStatus) bool {
	for _, valid := range ValidStatuses {
		if s == valid {
			return true
		}
	}
	return false
}

// Environment represents a tracked environment in the state database.
type Environment struct {
	ID         string            // 32 hex chars
	Backend    string            // Backend type (e.g., "worktree")
	BackendID  string            // Backend-specific identifier (may be empty)
	RepoPath   string            // Path to the original repository
	RemoteURL  string            // Git remote URL (may be empty)
	BranchName string            // Branch name (env/<short-id>)
	BaseBranch string            // Branch environment was created from
	CreatedAt  time.Time         // When environment was created
	Status     EnvironmentStatus // Current status
}

// ErrEnvironmentNotFound is returned when an environment with the given ID does not exist.
var ErrEnvironmentNotFound = errors.New("environment not found")

// ErrAmbiguousPrefix is returned when an ID prefix matches multiple environments.
var ErrAmbiguousPrefix = errors.New("ambiguous environment ID prefix")

// AmbiguousPrefixError is returned when an ID prefix matches multiple environments.
// It includes the list of matching environments for better error messages.
type AmbiguousPrefixError struct {
	Prefix  string
	Matches []*Environment
}

func (e *AmbiguousPrefixError) Error() string {
	return fmt.Sprintf("%s: '%s' matches %d environments", ErrAmbiguousPrefix.Error(), e.Prefix, len(e.Matches))
}

func (e *AmbiguousPrefixError) Unwrap() error {
	return ErrAmbiguousPrefix
}

// ErrInvalidPrefix is returned when an ID prefix contains non-hex characters.
var ErrInvalidPrefix = errors.New("invalid ID prefix: must contain only hexadecimal characters")

// ErrInvalidStatus is returned when an invalid status is provided.
var ErrInvalidStatus = errors.New("invalid status")

// isHexString returns true if s contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// CreateEnvironment inserts a new environment into the database.
func (db *DB) CreateEnvironment(env *Environment) error {
	if !IsValidStatus(env.Status) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, env.Status)
	}

	_, err := db.Exec(`
		INSERT INTO environments (
			id, backend, backend_id, repo_path, remote_url,
			branch_name, base_branch, created_at, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		env.ID,
		env.Backend,
		nullString(env.BackendID),
		env.RepoPath,
		nullString(env.RemoteURL),
		env.BranchName,
		env.BaseBranch,
		env.CreatedAt.UTC().Format(time.RFC3339),
		string(env.Status),
	)
	if err != nil {
		return fmt.Errorf("failed to create environment: %w", err)
	}
	return nil
}

// GetEnvironment retrieves an environment by full ID.
func (db *DB) GetEnvironment(id string) (*Environment, error) {
	row := db.QueryRow(`
		SELECT id, backend, backend_id, repo_path, remote_url,
		       branch_name, base_branch, created_at, status
		FROM environments WHERE id = ?`, id)

	env, err := scanEnvironment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEnvironmentNotFound
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}
	return env, nil
}

// GetEnvironmentByPrefix retrieves an environment by ID prefix.
// Returns ErrEnvironmentNotFound if no match, ErrAmbiguousPrefix if multiple matches,
// or ErrInvalidPrefix if the prefix contains non-hex characters.
func (db *DB) GetEnvironmentByPrefix(prefix string) (*Environment, error) {
	if prefix == "" || !isHexString(prefix) {
		return nil, ErrInvalidPrefix
	}

	rows, err := db.Query(`
		SELECT id, backend, backend_id, repo_path, remote_url,
		       branch_name, base_branch, created_at, status
		FROM environments WHERE id LIKE ? || '%'`, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to query environments: %w", err)
	}
	defer rows.Close()

	var envs []*Environment
	for rows.Next() {
		env, err := scanEnvironment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		envs = append(envs, env)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating environments: %w", err)
	}

	switch len(envs) {
	case 0:
		return nil, ErrEnvironmentNotFound
	case 1:
		return envs[0], nil
	default:
		return nil, &AmbiguousPrefixError{Prefix: prefix, Matches: envs}
	}
}

// UpdateEnvironment updates an existing environment.
func (db *DB) UpdateEnvironment(env *Environment) error {
	if !IsValidStatus(env.Status) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, env.Status)
	}

	result, err := db.Exec(`
		UPDATE environments SET
			backend = ?,
			backend_id = ?,
			repo_path = ?,
			remote_url = ?,
			branch_name = ?,
			base_branch = ?,
			status = ?
		WHERE id = ?`,
		env.Backend,
		nullString(env.BackendID),
		env.RepoPath,
		nullString(env.RemoteURL),
		env.BranchName,
		env.BaseBranch,
		string(env.Status),
		env.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update environment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrEnvironmentNotFound
	}
	return nil
}

// DeleteEnvironment removes an environment from the database.
func (db *DB) DeleteEnvironment(id string) error {
	result, err := db.Exec("DELETE FROM environments WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrEnvironmentNotFound
	}
	return nil
}

// ListOptions specifies filters for listing environments.
type ListOptions struct {
	RepoPath string              // Filter by repository path (exact match)
	Backend  string              // Filter by backend name
	Statuses []EnvironmentStatus // Filter by status (any of these)
}

// ListEnvironments returns all environments matching the given filters.
// If no filters are specified, returns all environments.
func (db *DB) ListEnvironments(opts ListOptions) ([]*Environment, error) {
	query := `
		SELECT id, backend, backend_id, repo_path, remote_url,
		       branch_name, base_branch, created_at, status
		FROM environments
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
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var envs []*Environment
	for rows.Next() {
		env, err := scanEnvironment(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		envs = append(envs, env)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating environments: %w", err)
	}

	return envs, nil
}

// CountEnvironments returns the number of environments matching the given filters.
func (db *DB) CountEnvironments(opts ListOptions) (int, error) {
	query := "SELECT COUNT(*) FROM environments"

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
		return 0, fmt.Errorf("failed to count environments: %w", err)
	}

	return count, nil
}

// scanner is an interface for sql.Row and sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanEnvironment scans a row into an Environment struct.
func scanEnvironment(s scanner) (*Environment, error) {
	var env Environment
	var backendID, remoteURL sql.NullString
	var createdAt string

	err := s.Scan(
		&env.ID,
		&env.Backend,
		&backendID,
		&env.RepoPath,
		&remoteURL,
		&env.BranchName,
		&env.BaseBranch,
		&createdAt,
		&env.Status,
	)
	if err != nil {
		return nil, err
	}

	env.BackendID = backendID.String
	env.RemoteURL = remoteURL.String

	env.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return &env, nil
}

// nullString converts an empty string to sql.NullString for optional fields.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
