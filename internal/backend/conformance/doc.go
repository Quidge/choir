// Package conformance provides backend-agnostic conformance tests that verify
// backends correctly implement the Backend interface contract.
//
// These tests would have caught bugs like issue #46 where the validation layer
// rejected relative file mount target paths that the worktree backend handled correctly.
//
// # Running Conformance Tests
//
// Conformance tests are gated behind build tags and do not run with regular `go test`.
//
// Run worktree backend conformance tests:
//
//	go test -tags=conformance,worktree ./internal/backend/conformance
//
// Run all conformance tests (when more backends are available):
//
//	go test -tags=conformance,worktree,lima ./internal/backend/conformance
//
// # Adding a New Backend
//
// To add conformance tests for a new backend:
//
//  1. Create a new test file (e.g., lima_test.go) with appropriate build tags:
//
//     //go:build conformance && lima
//
//  2. Register the backend and create a test function:
//
//     func TestLimaConformance(t *testing.T) {
//     be, _ := backend.Get(backend.BackendConfig{Type: "lima"})
//     suite := &ConformanceSuite{Backend: be, RepoSetup: SetupGitRepo}
//     suite.Run(t)
//     }
//
// # Test Categories
//
// The conformance suite tests:
//   - Lifecycle: Create, Destroy, Status, Exec operations
//   - FileMounts: Relative/absolute paths, readonly/writable, directories
//   - Environment: Environment variable handling and escaping
//   - SetupCommands: Command execution order, working directory, failure handling
package conformance
