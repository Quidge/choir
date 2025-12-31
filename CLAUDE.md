# CLAUDE.md

## References
- @README.md

## Testing

```bash
# Run unit tests
go test ./...

# Run backend conformance tests (requires build tags)
go test -tags=conformance,worktree ./internal/backend/conformance
```

Conformance tests verify backends correctly implement the `Backend` interface contract. They're gated behind build tags so they don't run with regular `go test`.
