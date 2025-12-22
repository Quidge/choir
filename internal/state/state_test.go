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

	// Verify agents table exists with expected columns
	_, err = db.Exec(`
		INSERT INTO agents (task_id, backend, repo_path, branch_name, base_branch, created_at, status)
		VALUES ('test', 'local', '/test', 'branch', 'main', '2024-01-01T00:00:00Z', 'running')
	`)
	if err != nil {
		t.Errorf("failed to insert into agents table: %v", err)
	}
}

func TestCRUD(t *testing.T) {
	db := openTestDB(t)

	now := time.Now().Truncate(time.Second)
	agent := &Agent{
		TaskID:     "task-123",
		Backend:    "local",
		BackendID:  "lima-abc",
		RepoPath:   "/home/user/project",
		RemoteURL:  "git@github.com:user/project.git",
		BranchName: "agent/feature-x",
		BaseBranch: "main",
		CreatedAt:  now,
		Status:     StatusRunning,
		Prompt:     "Implement feature X",
		Notes:      "Some notes",
	}

	t.Run("Create", func(t *testing.T) {
		err := db.CreateAgent(agent)
		if err != nil {
			t.Fatalf("CreateAgent() failed: %v", err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		got, err := db.GetAgent("task-123")
		if err != nil {
			t.Fatalf("GetAgent() failed: %v", err)
		}

		if got.TaskID != agent.TaskID {
			t.Errorf("TaskID = %q, want %q", got.TaskID, agent.TaskID)
		}
		if got.Backend != agent.Backend {
			t.Errorf("Backend = %q, want %q", got.Backend, agent.Backend)
		}
		if got.BackendID != agent.BackendID {
			t.Errorf("BackendID = %q, want %q", got.BackendID, agent.BackendID)
		}
		if got.RepoPath != agent.RepoPath {
			t.Errorf("RepoPath = %q, want %q", got.RepoPath, agent.RepoPath)
		}
		if got.RemoteURL != agent.RemoteURL {
			t.Errorf("RemoteURL = %q, want %q", got.RemoteURL, agent.RemoteURL)
		}
		if got.BranchName != agent.BranchName {
			t.Errorf("BranchName = %q, want %q", got.BranchName, agent.BranchName)
		}
		if got.BaseBranch != agent.BaseBranch {
			t.Errorf("BaseBranch = %q, want %q", got.BaseBranch, agent.BaseBranch)
		}
		if !got.CreatedAt.Equal(agent.CreatedAt) {
			t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, agent.CreatedAt)
		}
		if got.Status != agent.Status {
			t.Errorf("Status = %q, want %q", got.Status, agent.Status)
		}
		if got.Prompt != agent.Prompt {
			t.Errorf("Prompt = %q, want %q", got.Prompt, agent.Prompt)
		}
		if got.Notes != agent.Notes {
			t.Errorf("Notes = %q, want %q", got.Notes, agent.Notes)
		}
	})

	t.Run("Get not found", func(t *testing.T) {
		_, err := db.GetAgent("nonexistent")
		if !errors.Is(err, ErrAgentNotFound) {
			t.Errorf("GetAgent(nonexistent) error = %v, want ErrAgentNotFound", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		agent.Status = StatusStopped
		agent.Notes = "Updated notes"

		err := db.UpdateAgent(agent)
		if err != nil {
			t.Fatalf("UpdateAgent() failed: %v", err)
		}

		got, err := db.GetAgent("task-123")
		if err != nil {
			t.Fatalf("GetAgent() after update failed: %v", err)
		}

		if got.Status != StatusStopped {
			t.Errorf("Status = %q, want %q", got.Status, StatusStopped)
		}
		if got.Notes != "Updated notes" {
			t.Errorf("Notes = %q, want %q", got.Notes, "Updated notes")
		}
	})

	t.Run("Update not found", func(t *testing.T) {
		notFound := &Agent{
			TaskID:     "nonexistent",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			Status:     StatusRunning,
		}
		err := db.UpdateAgent(notFound)
		if !errors.Is(err, ErrAgentNotFound) {
			t.Errorf("UpdateAgent(nonexistent) error = %v, want ErrAgentNotFound", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := db.DeleteAgent("task-123")
		if err != nil {
			t.Fatalf("DeleteAgent() failed: %v", err)
		}

		_, err = db.GetAgent("task-123")
		if !errors.Is(err, ErrAgentNotFound) {
			t.Errorf("GetAgent() after delete error = %v, want ErrAgentNotFound", err)
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		err := db.DeleteAgent("nonexistent")
		if !errors.Is(err, ErrAgentNotFound) {
			t.Errorf("DeleteAgent(nonexistent) error = %v, want ErrAgentNotFound", err)
		}
	})
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
		agent := &Agent{
			TaskID:     "test",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     "invalid",
		}

		err := db.CreateAgent(agent)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("CreateAgent() with invalid status error = %v, want ErrInvalidStatus", err)
		}
	})

	t.Run("update with invalid status", func(t *testing.T) {
		// First create a valid agent
		agent := &Agent{
			TaskID:     "test-status",
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     StatusRunning,
		}
		if err := db.CreateAgent(agent); err != nil {
			t.Fatalf("CreateAgent() failed: %v", err)
		}

		// Try to update with invalid status
		agent.Status = "invalid"
		err := db.UpdateAgent(agent)
		if !errors.Is(err, ErrInvalidStatus) {
			t.Errorf("UpdateAgent() with invalid status error = %v, want ErrInvalidStatus", err)
		}
	})
}

func TestOptionalFields(t *testing.T) {
	db := openTestDB(t)

	// Create agent with minimal fields (no optional fields)
	agent := &Agent{
		TaskID:     "minimal",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusProvisioning,
	}

	err := db.CreateAgent(agent)
	if err != nil {
		t.Fatalf("CreateAgent() failed: %v", err)
	}

	got, err := db.GetAgent("minimal")
	if err != nil {
		t.Fatalf("GetAgent() failed: %v", err)
	}

	// Optional fields should be empty strings
	if got.BackendID != "" {
		t.Errorf("BackendID = %q, want empty", got.BackendID)
	}
	if got.RemoteURL != "" {
		t.Errorf("RemoteURL = %q, want empty", got.RemoteURL)
	}
	if got.Prompt != "" {
		t.Errorf("Prompt = %q, want empty", got.Prompt)
	}
	if got.Notes != "" {
		t.Errorf("Notes = %q, want empty", got.Notes)
	}
}

func TestListAgents(t *testing.T) {
	db := openTestDB(t)

	// Create test agents
	agents := []*Agent{
		{
			TaskID:     "task-1",
			Backend:    "local",
			RepoPath:   "/project-a",
			BranchName: "agent/1",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-3 * time.Hour),
			Status:     StatusRunning,
		},
		{
			TaskID:     "task-2",
			Backend:    "local",
			RepoPath:   "/project-a",
			BranchName: "agent/2",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			Status:     StatusStopped,
		},
		{
			TaskID:     "task-3",
			Backend:    "cloud",
			RepoPath:   "/project-b",
			BranchName: "agent/3",
			BaseBranch: "main",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			Status:     StatusRunning,
		},
		{
			TaskID:     "task-4",
			Backend:    "cloud",
			RepoPath:   "/project-b",
			BranchName: "agent/4",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     StatusFailed,
		},
	}

	for _, a := range agents {
		if err := db.CreateAgent(a); err != nil {
			t.Fatalf("CreateAgent(%s) failed: %v", a.TaskID, err)
		}
	}

	t.Run("list all", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 4 {
			t.Errorf("len(ListAgents()) = %d, want 4", len(got))
		}

		// Should be ordered by created_at DESC (newest first)
		if got[0].TaskID != "task-4" {
			t.Errorf("first agent = %q, want task-4", got[0].TaskID)
		}
	})

	t.Run("filter by repo_path", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{RepoPath: "/project-a"})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListAgents(repo=/project-a)) = %d, want 2", len(got))
		}

		for _, a := range got {
			if a.RepoPath != "/project-a" {
				t.Errorf("agent %s has RepoPath = %q, want /project-a", a.TaskID, a.RepoPath)
			}
		}
	})

	t.Run("filter by backend", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{Backend: "cloud"})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListAgents(backend=cloud)) = %d, want 2", len(got))
		}

		for _, a := range got {
			if a.Backend != "cloud" {
				t.Errorf("agent %s has Backend = %q, want cloud", a.TaskID, a.Backend)
			}
		}
	})

	t.Run("filter by single status", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{Statuses: []Status{StatusRunning}})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListAgents(status=running)) = %d, want 2", len(got))
		}

		for _, a := range got {
			if a.Status != StatusRunning {
				t.Errorf("agent %s has Status = %q, want running", a.TaskID, a.Status)
			}
		}
	})

	t.Run("filter by multiple statuses", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{Statuses: []Status{StatusStopped, StatusFailed}})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("len(ListAgents(status=stopped,failed)) = %d, want 2", len(got))
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{
			Backend:  "local",
			Statuses: []Status{StatusRunning},
		})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("len(ListAgents(backend=local, status=running)) = %d, want 1", len(got))
		}

		if len(got) > 0 && got[0].TaskID != "task-1" {
			t.Errorf("agent = %q, want task-1", got[0].TaskID)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		got, err := db.ListAgents(ListOptions{RepoPath: "/nonexistent"})
		if err != nil {
			t.Fatalf("ListAgents() failed: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("len(ListAgents(repo=/nonexistent)) = %d, want 0", len(got))
		}
	})
}

func TestCountAgents(t *testing.T) {
	db := openTestDB(t)

	// Create test agents
	for i, status := range []Status{StatusRunning, StatusRunning, StatusStopped} {
		agent := &Agent{
			TaskID:     string(rune('a' + i)),
			Backend:    "local",
			RepoPath:   "/test",
			BranchName: "test",
			BaseBranch: "main",
			CreatedAt:  time.Now(),
			Status:     status,
		}
		if err := db.CreateAgent(agent); err != nil {
			t.Fatalf("CreateAgent() failed: %v", err)
		}
	}

	t.Run("count all", func(t *testing.T) {
		count, err := db.CountAgents(ListOptions{})
		if err != nil {
			t.Fatalf("CountAgents() failed: %v", err)
		}
		if count != 3 {
			t.Errorf("CountAgents() = %d, want 3", count)
		}
	})

	t.Run("count with filter", func(t *testing.T) {
		count, err := db.CountAgents(ListOptions{Statuses: []Status{StatusRunning}})
		if err != nil {
			t.Fatalf("CountAgents() failed: %v", err)
		}
		if count != 2 {
			t.Errorf("CountAgents(status=running) = %d, want 2", count)
		}
	})
}

func TestConcurrentReads(t *testing.T) {
	db := openTestDB(t)

	// Create a test agent
	agent := &Agent{
		TaskID:     "concurrent-test",
		Backend:    "local",
		RepoPath:   "/test",
		BranchName: "test",
		BaseBranch: "main",
		CreatedAt:  time.Now(),
		Status:     StatusRunning,
	}
	if err := db.CreateAgent(agent); err != nil {
		t.Fatalf("CreateAgent() failed: %v", err)
	}

	// Perform concurrent reads, collecting errors via channel
	// (t.Errorf is not safe to call from goroutines)
	const numGoroutines = 10
	errs := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := db.GetAgent("concurrent-test")
			errs <- err
		}()
	}

	// Collect results and check for errors
	for i := 0; i < numGoroutines; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent GetAgent() failed: %v", err)
		}
	}
}
