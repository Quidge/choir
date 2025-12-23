# CLI Usage Guide

> **Note**: This document reflects the upcoming `choir env` command structure. See [#31](https://github.com/Quidge/choir/issues/31) for the design specification.

This guide covers common workflows for using choir to manage parallel development environments.

## Quick Start

```bash
# Create a new environment
choir env create --base main
# Returns: a1b2c3d4e5f6

# Attach to it
choir env attach a1b2c3d4e5f6

# In another terminal, create a second environment
choir env create --base main
# Returns: x9y8z7w6v5u4

# See all environments
choir env list

# When done, remove an environment
choir env rm a1b2
```

## Commands

### choir env create

Create a new isolated development environment.

```bash
# Basic usage - creates environment from current branch
choir env create

# Create from a specific branch
choir env create --base main

# Skip setup commands from .choir.yaml
choir env create --no-setup
```

The create command:
1. Generates a unique environment ID (32 hex chars, displays 12)
2. Creates a worktree at `<repo-parent>/choir-<short-id>/`
3. Creates a new branch `env/<short-id>` from the base branch
4. Runs any setup commands defined in `.choir.yaml`
5. Prints the environment ID (does NOT enter the environment)

### choir env attach

Enter an existing environment's shell.

```bash
# Attach by full ID
choir env attach a1b2c3d4e5f6

# Attach by unique prefix
choir env attach a1b2
```

Use this to enter an environment and start working. You can run `claude` or other tools once inside.

### choir env list

Show all environments.

```bash
# List all active environments
choir env list

# Include removed/failed environments
choir env list --all

# Filter to current repository only
choir env list --repo

# Filter by backend
choir env list --backend worktree
```

Example output:
```
ID            STATUS  BRANCH              CREATED
a1b2c3d4e5f6  ready   env/a1b2c3d4e5f6   5m ago
x9y8z7w6v5u4  ready   env/x9y8z7w6v5u4   2h ago
```

### choir env rm

Remove an environment and its worktree.

```bash
# Remove by ID or prefix
choir env rm a1b2

# Force removal without confirmation
choir env rm a1b2 -f
```

This destroys the worktree directory and marks the environment as removed. Any uncommitted changes in the worktree will be lost.

### choir env status

Show detailed information about an environment.

```bash
choir env status a1b2
```

Example output:
```
ID:       a1b2c3d4e5f6789...
Status:   ready
Backend:  worktree
Branch:   env/a1b2c3d4e5f6
Base:     main
Repo:     /Users/me/projects/myapp
Remote:   git@github.com:user/myapp.git
Created:  2025-01-15 10:30:00 (2 hours ago)
Path:     /Users/me/projects/choir-a1b2c3d4e5f6
```

## Workflows

### Parallel Feature Development

Work on multiple features simultaneously without branch switching:

```bash
# Terminal 1: Work on authentication
ENV1=$(choir env create --base main)
choir env attach $ENV1
# Now in /path/to/choir-<id> on branch env/<id>
# Start claude, make changes, commit as needed

# Terminal 2: Work on API refactor
ENV2=$(choir env create --base main)
choir env attach $ENV2
# Work independently from Terminal 1

# Check status from main repo
cd /path/to/main-repo
choir env list
```

### Bug Fix While Feature In Progress

Pause feature work to fix an urgent bug:

```bash
# You're working on a feature
ENV=$(choir env create --base main)
choir env attach $ENV

# Urgent bug comes in - create another environment
# (In a new terminal)
HOTFIX=$(choir env create --base main)
choir env attach $HOTFIX

# Fix the bug, commit, push
git add . && git commit -m "Fix crash"
git push origin env/<id>

# Remove the hotfix environment when done
choir env rm $HOTFIX

# Continue feature work in original terminal
```

### Code Review Workflow

Review a PR while keeping your work intact:

```bash
# Your current work
ENV=$(choir env create --base main)
choir env attach $ENV

# Review someone's PR (in new terminal)
REVIEW=$(choir env create --base feature/their-branch)
choir env attach $REVIEW

# Examine their code, run tests
go test ./...

# Clean up when done reviewing
choir env rm $REVIEW
```

### Experiment Safely

Try risky changes without affecting your main work:

```bash
# Create an experimental environment
EXPERIMENT=$(choir env create --base main)
choir env attach $EXPERIMENT

# Try things out - if it doesn't work, just remove it
choir env rm $EXPERIMENT

# Nothing in your main repo was affected
```

## Tips

### Working with IDs

Environment IDs are auto-generated hex strings:
- Full ID: 32 characters (stored in database)
- Display ID: 12 characters (shown in output)
- Prefix matching: Use any unique prefix (e.g., `a1b2`)

```bash
# These all work if "a1b2" is a unique prefix
choir env attach a1b2c3d4e5f6
choir env attach a1b2c3d4
choir env attach a1b2
```

### Scripting

```bash
# Create and immediately attach
ENV_ID=$(choir env create --base main)
choir env attach $ENV_ID

# Batch cleanup - remove all environments for this repo
choir env list --repo | tail -n +2 | awk '{print $1}' | xargs -I{} choir env rm {} -f
```

### Finding Your Environments

```bash
# From anywhere, list all environments
choir env list

# Use status for detailed info
choir env status a1b2
```

### Cleaning Up

Regularly clean up finished environments:

```bash
# See all environments including removed ones
choir env list --all

# Remove environments you're done with
choir env rm <id1>
choir env rm <id2>
```

### Git Operations

Each environment has its own branch. Standard git operations work:

```bash
# Inside an environment
git status
git add .
git commit -m "Progress on feature"
git push origin env/<id>

# Create a PR from the environment branch
gh pr create --base main
```

## Configuration

Create a `.choir.yaml` in your repository root to configure environment setup:

```yaml
version: 1

# Commands to run when creating a new environment
setup:
  - npm install
  - cp .env.example .env

# Environment variables
env:
  NODE_ENV: development
```

## Troubleshooting

### "environment not found"

No environment matches the given ID/prefix. Check available environments:
```bash
choir env list --all
```

### "ambiguous prefix"

Multiple environments match the prefix. Use a longer prefix:
```bash
choir env list  # See full IDs
choir env attach a1b2c3  # Use more characters
```

### "not in a git repository"

Run choir commands from within a git repository.

### "cannot create from detached HEAD"

You're on a detached HEAD. Specify a base branch:
```bash
choir env create --base main
```

### Environment shows "failed" status

The environment was created but setup didn't complete. Check what went wrong and remove it:
```bash
choir env status <id>  # See details
choir env rm <id>
choir env create --base main  # Try again
```
