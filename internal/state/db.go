package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB connection to the state database.
type DB struct {
	*sql.DB
	path string
}

// DefaultDBPath returns the default database path (~/.local/share/choir/state.db).
func DefaultDBPath() (string, error) {
	// Follow XDG Base Directory spec: use $XDG_DATA_HOME or ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}
	return filepath.Join(dataHome, "choir", "state.db"), nil
}

// Open opens or creates the state database at the given path.
// Use ":memory:" for an in-memory database (useful for testing).
// If path is empty, uses DefaultDBPath().
func Open(path string) (*DB, error) {
	var err error
	if path == "" {
		path, err = DefaultDBPath()
		if err != nil {
			return nil, err
		}
	}

	// Create parent directory if using a file-based database
	if path != ":memory:" {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}

	// Build DSN with appropriate options
	var dsn string
	if path == ":memory:" {
		// For in-memory databases, use shared cache mode so multiple connections
		// access the same database. This is important for concurrent reads.
		dsn = "file::memory:?cache=shared"
	} else {
		// For file-based databases, use WAL mode for better concurrent read performance
		dsn = fmt.Sprintf("file:%s?_journal_mode=WAL", path)
	}

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// For in-memory databases with shared cache, limit connections to avoid issues
	if path == ":memory:" {
		sqlDB.SetMaxOpenConns(1)
	}

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db := &DB{
		DB:   sqlDB,
		path: path,
	}

	// Run migrations to ensure schema is up to date
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Path returns the database file path, or ":memory:" for in-memory databases.
func (db *DB) Path() string {
	return db.path
}
