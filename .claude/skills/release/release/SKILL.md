---
name: release
description: Create releases and bump versions. Use when the user wants to release, bump version, or create a tag.
---

# Release Manager

Create and manage releases for choir.

## When to Use This Skill

- "Release a new version"
- "Bump the version"
- "Create a patch release"
- "I want to release v0.1.0"
- "Push a new tag"
- "Make a minor release"

## Instructions

### Creating a Release

1. **Determine the bump type** based on what changed:
   - **major**: Breaking changes (removed commands, changed behavior)
   - **minor**: New features (new commands, new flags, new backends)
   - **patch**: Bug fixes, improvements, dependency updates

2. **Run the release script**:
   ```bash
   ./scripts/release.sh [major|minor|patch]
   ```

3. **Confirm the release** when prompted (or use `--force` to skip)

4. **Monitor the release workflow** at https://github.com/Quidge/choir/actions

### Choosing Version Type

| Changed | Bump |
|---------|------|
| Breaking change to CLI or config format | `major` |
| New command, flag, or backend | `minor` |
| Bug fix | `patch` |
| Documentation only | `patch` |
| Dependency update | `patch` |

### Pre-Release Verification

Before releasing, the script automatically checks:
- Must be on `main` branch
- Local must be up-to-date with `origin/main`

Optionally verify:
```bash
go test ./...
go test -tags=conformance,worktree ./internal/backend/conformance
go build -o choir .
```

### What Happens After Release

The GitHub Actions workflow will:
1. Build binaries for macOS and Linux (arm64, amd64)
2. Create GitHub release with artifacts
3. Update Homebrew tap (Quidge/homebrew-choir)
