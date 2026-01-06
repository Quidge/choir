package state

import (
	"errors"
	"testing"
	"time"
)

// openTestDB creates an in-memory database for testing.
func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpen(t *testing.T) {
	t.Run("in-memory database", func(t *testing.T) {
		db, err := Open(":memory:")
		if err != nil {
			t.Fatalf("Open(:memory:) failed: %v", err)
		}
		defer db.Close()

		if db.Path() != ":memory:" {
			t.Errorf("Path() = %q, want %q", db.Path(), ":memory:")
		}
	})

	t.Run("temp file database", func(t *testing.T) {
		path := t.TempDir() + "/test.db"
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open(%q) failed: %v", path, err)
		}
		defer db.Close()

		if db.Path() != path {
			t.Errorf("Path() = %q, want %q", db.Path(), path)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		path := t.TempDir() + "/nested/dirs/test.db"
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open(%q) failed: %v", path, err)
		}
		defer db.Close()
	})
}

func TestMigrations(t *testing.T) {
	db := openTestDB(t)

	version, err := db.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion() failed: %v", err)
	}

	expectedVersion := len(migrations)
	if version != expectedVersion {
		t.Errorf("SchemaVersion() = %d, want %d", version, expectedVersion)
	}

	// Verify environments table exists with expected columns
	_, err = db.Exec(`
		INSERT INTO environments (id, backend, repo_path, branch_name, base_branch, created_at, status)
		VALUES ('abc123def456abc123def456abc12345', 'local', '/test', 'branch', 'main', '2024-01-01T00:00:00Z', 'ready')
	`)
	if err != nil {
		t.Errorf("failed to insert into environments table: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID() failed: %v", err)
	}

	if len(id) != IDLength {
		t.Errorf("len(GenerateID()) = %d, want %d", len(id), IDLength)
	}

	// Verify it's hex
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("GenerateID() contains non-hex character: %c", c)
		}
	}

	// Verify uniqueness
	id2, _ := GenerateID()
	if id == id2 {
		t.Error("GenerateID() returned same ID twice")
	}
}

func TestShortID(t *testing.T) {
	id := "abc123def456abc123def456abc12345"
	short := ShortID(id)

	if len(short) != ShortIDLength {
		t.Errorf("len(ShortID()) = %d, want %d", len(short), ShortIDLength)
	}

	if short != "abc123def456" {
		t.Errorf("ShortID() = %q, want %q", short, "abc123def456")
	}

	// Test with short input
	shortInput := "abc"
	if ShortID(shortInput) != shortInput {
		t.Errorf("ShortID(%q) = %q, want %q", shortInput, ShortID(shortInput), shortInput)
	}
}

func TestCRUD(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	env := &Environment{
		ID:         "abc123def456abc123def456abc12345",
		Backend:    "local",
		BackendID:  "/path/to/worktree",
		RepoPath:   "/home/user/project",
		RemoteURL:  "git@github.com:user/project.git",
		BranchName: "env/abc123def456",
		BaseBranch: "main",
		CreatedAt:  now,
		Status:     StatusReady,
	}

	t.Run("Create", func(t *testing.T) {
		err := db.CreateEnvironment(env)
		if err != nil {
			t.Fatalf("CreateEnvironment() failed: %v", err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		got, err := db.GetEnvironment("abc123def456abc123def456abc12345")
		if err != nil {
			t.Fatalf("GetEnvironment() failed: %v", err)
		}

		if got.ID != env.ID {
			t.Errorf("ID = %q, want %q", got.ID, env.ID)
		}
		if got.Backend != env.Backend {
			t.Errorf("Backend = %q, want %q", got.Backend, env.Backend)
		}
		if got.BackendID != env.BackendID {
			t.Errorf("BackendID = %q, want %q", got.BackendID, env.BackendID)
		}
		if got.RepoPath != env.RepoPath {
			t.Errorf("RepoPath = %q, want %q", got.RepoPath, env.RepoPath)
		}
		if got.RemoteURL != env.RemoteURL {
			t.Errorf("RemoteURL = %q, want %q", got.RemoteURL, env.RemoteURL)
		}
		if got.BranchName != env.BranchName {
			t.Errorf("BranchName = %q, want %q", got.BranchName, env.BranchName)
		}
		if got.BaseBranch != env.BaseBranch {
			t.Errorf("BaseBranch = %q, want %q", got.BaseBranch, env.BaseBranch)
		}
		if !got.CreatedAt.Equal(env.CreatedAt) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, env.CreatedAt)
		}
		if got.Status != env.Status {
			t.Errorf("Status = %q, want %q", got.Status, env.Status)
		}
	})

	t.Run("Get not found", func(t *testing.T) {
		_, err := db.GetEnvironment("nonexistent")
		if !errors.Is(err, ErrEnvironmentNotFound) {
			t.Errorf("GetEnvironment(nonexistent) error = %v, want ErrEnvironmentNotFound", err)
		}
	})

	t.Run("GetByPrefix", func(t *testing.T) {
		got, err := db.GetEnvironmentByPrefix("abc123")
		if err != nil {
			t.Fatalf("GetEnvironmentByPrefix() failed: %v", err)
		}
		if got.ID != env.ID {
			t.Errorf("ID = %q, want %q", got.ID, env.ID)
		}
	})

	t.Run("GetByPrefix not found", func(t *testing.T) {
		_, err := db.GetEnvironmentByPrefix("fff999")
		if !errors.Is(err, ErrEnvironmentNotFound) {
			t.Errorf("GetEnvironmentByPrefix(fff999) error = %v, want ErrEnvironmentNotFound", err)
		}
	})

	t.Run("GetByPrefix invalid characters", func(t *testing.T) {
		_, err := db.GetEnvironmentByPrefix("abc%")
		if !errors.Is(err, ErrInvalidPrefix) {
			t.Errorf("GetEnvironmentByPrefix(abc%%) error = %v, want ErrInvalidPrefix", err)
		}
	})

	t.Run("GetByPrefix empty", func(t *testing.T) {
		_, err := db.GetEnvironmentByPrefix("")
		if !errors.Is(err, ErrInvalidPrefix) {
			t.Errorf("GetEnvironmentByPrefix(\"\") error = %v, want ErrInvalidPrefix", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		env.Status = StatusRemoved

		err := db.UpdateEnvironment(env)
		if err != nil {
			t.Fatalf("UpdateEnvironment() failed: %v", err)
		}

		got, err := db.GetEnvironment("abc123def456abc123def456abc12345")
		if err != nil {
			t.Fatalf("GetEnvironment() after update failed: %v", err)
		}

		if got.Status != StatusRemoved {
			t.Errorf("Status = %q, want %q", got.Status, StatusRemoved)
		}
	})

	t.Run("Update not found", func(t *testing.T) {
		notFound := &Environment{
			ID:         "nonexistent12345678901234567890",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			Status:     StatusReady,
		}
		err := db.UpdateEnvironment(notFound)
		if !errors.Is(err, ErrEnvironmentNotFound) {
			t.Errorf("UpdateEnvironment(nonexistent) error = %v, want ErrEnvironmentNotFound", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.DeleteEnvironment("abc123def456abc123def456abc12345")
		if err != nil {
			t.Fatalf("DeleteEnvironment() failed: %v", err)
		}

		_, err = db.GetEnvironment("abc123def456abc123def456abc12345")
		if !errors.Is(err, ErrEnvironmentNotFound) {
			t.Errorf("GetEnvironment() after delete error = %v, want ErrEnvironmentNotFound", err)
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		err := db.DeleteEnvironment("nonexistent")
		if !errors.Is(err, ErrEnvironmentNotFound) {
			t.Errorf("DeleteEnvironment(nonexistent) error = %v, want ErrEnvironmentNotFound", err)
		}
	})
}

func TestGetByPrefixAmbiguous(t *testing.T) {
	db := openTestDB(t)

	// Create two environments with similar prefixes
	env1 := &Environment{
		ID:         "abc123def456abc123def456abc12345",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "env/abc123def456",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusReady,
	}
	env2 := &Environment{
		ID:         "abc123xyz789abc123xyz789abc12345",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "env/abc123xyz789",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusFailed,
	}

	if err := db.CreateEnvironment(env1); err != nil {
		t.Fatalf("CreateEnvironment() failed: %v", err)
	}
	if err := db.CreateEnvironment(env2); err != nil {
		t.Fatalf("CreateEnvironment() failed: %v", err)
	}

	// Should fail with ambiguous prefix
	_, err := db.GetEnvironmentByPrefix("abc123")
	if !errors.Is(err, ErrAmbiguousPrefix) {
		t.Errorf("GetEnvironmentByPrefix(abc123) error = %v, want ErrAmbiguousPrefix", err)
	}

	// Verify error contains match details
	var ambiguousErr *AmbiguousPrefixError
	if !errors.As(err, &ambiguousErr) {
		t.Fatalf("expected *AmbiguousPrefixError, got %T", err)
	}
	if ambiguousErr.Prefix != "abc123" {
		t.Errorf("Prefix = %q, want %q", ambiguousErr.Prefix, "abc123")
	}
	if len(ambiguousErr.Matches) != 2 {
		t.Errorf("len(Matches) = %d, want 2", len(ambiguousErr.Matches))
	}

	// Verify error message format
	errMsg := ambiguousErr.Error()
	if errMsg != "ambiguous environment ID prefix: 'abc123' matches 2 environments" {
		t.Errorf("Error() = %q, want %q", errMsg, "ambiguous environment ID prefix: 'abc123' matches 2 environments")
	}

	// Should succeed with unique prefix
	got, err := db.GetEnvironmentByPrefix("abc123def")
	if err != nil {
		t.Fatalf("GetEnvironmentByPrefix(abc123def) failed: %v", err)
	}
	if got.ID != env1.ID {
		t.Errorf("ID = %q, want %q", got.ID, env1.ID)
	}
}

func TestStatusValidation(t *testing.T) {
	db := openTestDB(t)

	t.Run("valid statuses", func(t *testing.T) {
		for _, status := range ValidStatuses {
			if !IsValidStatus(status) {
				t.Errorf("IsValidStatus(%q) = false, want true", status)
			}
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		if IsValidStatus("invalid") {
			t.Error("IsValidStatus(invalid) = true, want false")
		}
	})

	t.Run("create with invalid status", func(t *testing.T) {
		env := &Environment{
			ID:         "test123456789012345678901234567",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     "invalid",
		}

		err := db.CreateEnvironment(env)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("CreateEnvironment() with invalid status error = %v, want ErrInvalidStatus", err)
		}
	})

	t.Run("update with invalid status", func(t *testing.T) {
		// First create a valid environment
		env := &Environment{
			ID:         "status1234567890123456789012345",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     StatusReady,
		}
		if err := db.CreateEnvironment(env); err != nil {
			t.Fatalf("CreateEnvironment() failed: %v", err)
		}

		// Try to update with invalid status
		env.Status = "invalid"
		err := db.UpdateEnvironment(env)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("UpdateEnvironment() with invalid status error = %v, want ErrInvalidStatus", err)
		}
	})
}

func TestOptionalFields(t *testing.T) {
	db := openTestDB(t)

	// Create environment with minimal fields (no optional fields)
	env := &Environment{
		ID:         "minimal123456789012345678901234",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusProvisioning,
	}

	err := db.CreateEnvironment(env)
	if err != nil {
		t.Fatalf("CreateEnvironment() failed: %v", err)
	}

	got, err := db.GetEnvironment("minimal123456789012345678901234")
	if err != nil {
		t.Fatalf("GetEnvironment() failed: %v", err)
	}

	// Optional fields should be empty strings
	if got.BackendID != "" {
		t.Errorf("BackendID = %q, want empty", got.BackendID)
	}
	if got.RemoteURL != "" {
		t.Errorf("RemoteURL = %q, want empty", got.RemoteURL)
	}
}

func TestListEnvironments(t *testing.T) {
	db := openTestDB(t)

	// Create test environments
	envs := []*Environment{
		{
			ID:         "env1abc123456789012345678901234",
			Backend:    "local",
			RepoPath:   "/project-a",
			BranchName: "env/1",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-3 * time.Hour),
			Status:     StatusReady,
		},
		{
			ID:         "env2abc123456789012345678901234",
			Backend:    "local",
			RepoPath:   "/project-a",
			BranchName: "env/2",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			Status:     StatusProvisioning,
		},
		{
			ID:         "env3abc123456789012345678901234",
			Backend:    "cloud",
			RepoPath:   "/project-b",
			BranchName: "env/3",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			Status:     StatusReady,
		},
		{
			ID:         "env4abc123456789012345678901234",
			Backend:    "cloud",
			RepoPath:   "/project-b",
			BranchName: "env/4",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     StatusFailed,
		},
	}

	for _, e := range envs {
		if err := db.CreateEnvironment(e); err != nil {
			t.Fatalf("CreateEnvironment(%s) failed: %v", e.ID, err)
		}
	}

	t.Run("list all", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 4 {
			t.Errorf("len(ListEnvironments()) = %d, want 4", len(got))
		}

		// Should be ordered by created_at DESC (newest first)
		if got[0].ID != "env4abc123456789012345678901234" {
			t.Errorf("first environment = %q, want env4...", got[0].ID)
		}
	})

	t.Run("filter by repo_path", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{RepoPath: "/project-a"})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListEnvironments(repo=/project-a)) = %d, want 2", len(got))
		}

		for _, e := range got {
			if e.RepoPath != "/project-a" {
				t.Errorf("environment %s has RepoPath = %q, want /project-a", e.ID, e.RepoPath)
			}
		}
	})

	t.Run("filter by backend", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{Backend: "cloud"})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListEnvironments(backend=cloud)) = %d, want 2", len(got))
		}

		for _, e := range got {
			if e.Backend != "cloud" {
				t.Errorf("environment %s has Backend = %q, want cloud", e.ID, e.Backend)
			}
		}
	})

	t.Run("filter by single status", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{Statuses: []EnvironmentStatus{StatusReady}})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListEnvironments(status=ready)) = %d, want 2", len(got))
		}

		for _, e := range got {
			if e.Status != StatusReady {
				t.Errorf("environment %s has Status = %q, want ready", e.ID, e.Status)
			}
		}
	})

	t.Run("filter by multiple statuses", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{Statuses: []EnvironmentStatus{StatusProvisioning, StatusFailed}})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListEnvironments(status=provisioning,failed)) = %d, want 2", len(got))
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{
			Backend:  "local",
			Statuses: []EnvironmentStatus{StatusReady},
		})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("len(ListEnvironments(backend=local, status=ready)) = %d, want 1", len(got))
		}

		if len(got) > 0 && got[0].ID != "env1abc123456789012345678901234" {
			t.Errorf("environment = %q, want env1...", got[0].ID)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		got, err := db.ListEnvironments(ListOptions{RepoPath: "/nonexistent"})
		if err != nil {
			t.Fatalf("ListEnvironments() failed: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("len(ListEnvironments(repo=/nonexistent)) = %d, want 0", len(got))
		}
	})
}

func TestCountEnvironments(t *testing.T) {
	db := openTestDB(t)

	// Create test environments
	for i, status := range []EnvironmentStatus{StatusReady, StatusReady, StatusProvisioning} {
		env := &Environment{
			ID:         string(rune('a'+i)) + "bc123456789012345678901234567",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     status,
		}
		if err := db.CreateEnvironment(env); err != nil {
			t.Fatalf("CreateEnvironment() failed: %v", err)
		}
	}

	t.Run("count all", func(t *testing.T) {
		count, err := db.CountEnvironments(ListOptions{})
		if err != nil {
			t.Fatalf("CountEnvironments() failed: %v", err)
		}
		if count != 3 {
			t.Errorf("CountEnvironments() = %d, want 3", count)
		}
	})

	t.Run("count with filter", func(t *testing.T) {
		count, err := db.CountEnvironments(ListOptions{Statuses: []EnvironmentStatus{StatusReady}})
		if err != nil {
			t.Fatalf("CountEnvironments() failed: %v", err)
		}
		if count != 2 {
			t.Errorf("CountEnvironments(status=ready) = %d, want 2", count)
		}
	})
}

func TestConcurrentReads(t *testing.T) {
	db := openTestDB(t)

	// Create a test environment
	env := &Environment{
		ID:         "concurrent12345678901234567890",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusReady,
	}
	if err := db.CreateEnvironment(env); err != nil {
		t.Fatalf("CreateEnvironment() failed: %v", err)
	}

	// Perform concurrent reads, collecting errors via channel
	// (t.Errorf is not safe to call from goroutines)
	const numGoroutines = 10
	errs := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := db.GetEnvironment("concurrent12345678901234567890")
			errs <- err
		}()
	}

	// Collect results and check for errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent GetEnvironment() failed: %v", err)
		}
	}
}
