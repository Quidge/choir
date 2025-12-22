package state

import (
	"database/sql"
	"errors"
	"fmt"
)

// migration represents a database schema migration.
type migration struct {
	version int
	name    string
	up      string
}

// migrations contains all database migrations in order.
// Add new migrations to the end of this slice.
var migrations = []migration{
	{
		version: 1,
		name:    "create_agents_table",
		up: `
CREATE TABLE agents (
    task_id       TEXT PRIMARY KEY,
    backend       TEXT NOT NULL,
    backend_id    TEXT,
    repo_path     TEXT NOT NULL,
    remote_url    TEXT,
    branch_name   TEXT NOT NULL,
    base_branch   TEXT NOT NULL,
    created_at    TEXT NOT NULL,
    status        TEXT NOT NULL,
    prompt        TEXT,
    notes         TEXT
);

CREATE INDEX idx_agents_repo ON agents(repo_path);
CREATE INDEX idx_agents_backend ON agents(backend);
CREATE INDEX idx_agents_status ON agents(status);
`,
	},
}

// migrate runs all pending migrations.
func (db *DB) migrate() error {
	// Create schema_migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get current schema version
	currentVersion, err := db.schemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// Run pending migrations
	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}

		if err := db.runMigration(m); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.version, m.name, err)
		}
	}

	return nil
}

// schemaVersion returns the current schema version, or 0 if no migrations have been applied.
func (db *DB) schemaVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// runMigration runs a single migration within a transaction.
func (db *DB) runMigration(m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run migration SQL
	if _, err := tx.Exec(m.up); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		m.version, m.name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// SchemaVersion returns the current schema version for external inspection.
func (db *DB) SchemaVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}
