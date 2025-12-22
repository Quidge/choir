package state

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Status represents the state of an agent.
type Status string

const (
	StatusProvisioning Status = "provisioning"
	StatusRunning      Status = "running"
	StatusStopped      Status = "stopped"
	StatusRemoved      Status = "removed"
	StatusFailed       Status = "failed"
)

// ValidStatuses contains all valid agent status values.
var ValidStatuses = []Status{
	StatusProvisioning,
	StatusRunning,
	StatusStopped,
	StatusRemoved,
	StatusFailed,
}

// IsValidStatus returns true if s is a valid status.
func IsValidStatus(s Status) bool {
	for _, valid := range ValidStatuses {
		if s == valid {
			return true
		}
	}
	return false
}

// Agent represents a tracked agent in the state database.
type Agent struct {
	TaskID     string
	Backend    string
	BackendID  string // May be empty
	RepoPath   string
	RemoteURL  string // May be empty
	BranchName string
	BaseBranch string
	CreatedAt  time.Time
	Status     Status
	Prompt     string // May be empty
	Notes      string // May be empty
}

// ErrAgentNotFound is returned when an agent with the given ID does not exist.
var ErrAgentNotFound = errors.New("agent not found")

// ErrInvalidStatus is returned when an invalid status is provided.
var ErrInvalidStatus = errors.New("invalid status")

// CreateAgent inserts a new agent into the database.
func (db *DB) CreateAgent(agent *Agent) error {
	if !IsValidStatus(agent.Status) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, agent.Status)
	}

	_, err := db.Exec(`
		INSERT INTO agents (
			task_id, backend, backend_id, repo_path, remote_url,
			branch_name, base_branch, created_at, status, prompt, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.TaskID,
		agent.Backend,
		nullString(agent.BackendID),
		agent.RepoPath,
		nullString(agent.RemoteURL),
		agent.BranchName,
		agent.BaseBranch,
		agent.CreatedAt.UTC().Format(time.RFC3339),
		string(agent.Status),
		nullString(agent.Prompt),
		nullString(agent.Notes),
	)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}
	return nil
}

// GetAgent retrieves an agent by task ID.
func (db *DB) GetAgent(taskID string) (*Agent, error) {
	row := db.QueryRow(`
		SELECT task_id, backend, backend_id, repo_path, remote_url,
		       branch_name, base_branch, created_at, status, prompt, notes
		FROM agents WHERE task_id = ?`, taskID)

	agent, err := scanAgent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgentNotFound
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return agent, nil
}

// UpdateAgent updates an existing agent.
func (db *DB) UpdateAgent(agent *Agent) error {
	if !IsValidStatus(agent.Status) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, agent.Status)
	}

	result, err := db.Exec(`
		UPDATE agents SET
			backend = ?,
			backend_id = ?,
			repo_path = ?,
			remote_url = ?,
			branch_name = ?,
			base_branch = ?,
			status = ?,
			prompt = ?,
			notes = ?
		WHERE task_id = ?`,
		agent.Backend,
		nullString(agent.BackendID),
		agent.RepoPath,
		nullString(agent.RemoteURL),
		agent.BranchName,
		agent.BaseBranch,
		string(agent.Status),
		nullString(agent.Prompt),
		nullString(agent.Notes),
		agent.TaskID,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrAgentNotFound
	}
	return nil
}

// DeleteAgent removes an agent from the database.
func (db *DB) DeleteAgent(taskID string) error {
	result, err := db.Exec("DELETE FROM agents WHERE task_id = ?", taskID)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrAgentNotFound
	}
	return nil
}

// scanner is an interface for sql.Row and sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanAgent scans a row into an Agent struct.
func scanAgent(s scanner) (*Agent, error) {
	var agent Agent
	var backendID, remoteURL, prompt, notes sql.NullString
	var createdAt string

	err := s.Scan(
		&agent.TaskID,
		&agent.Backend,
		&backendID,
		&agent.RepoPath,
		&remoteURL,
		&agent.BranchName,
		&agent.BaseBranch,
		&createdAt,
		&agent.Status,
		&prompt,
		&notes,
	)
	if err != nil {
		return nil, err
	}

	agent.BackendID = backendID.String
	agent.RemoteURL = remoteURL.String
	agent.Prompt = prompt.String
	agent.Notes = notes.String

	agent.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return &agent, nil
}

// nullString converts an empty string to sql.NullString for optional fields.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
