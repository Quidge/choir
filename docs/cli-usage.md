# CLI Usage Guide

This guide covers common workflows for using choir to manage parallel development environments.

## Quick Start

```bash
# Create a new environment and get its ID
choir env create
# Output: a1b2c3d4

# In another terminal, create a second environment
choir env create --base main
# Output: e5f6g7h8

# See all environments
choir env list

# Enter an environment's shell
choir env attach a1b2

# When done, remove an environment
choir env rm a1b2
```

## Commands

### env create

Create a new environment with a unique auto-generated ID.

```bash
# Basic usage - creates environment from current branch
choir env create

# Create from a specific branch
choir env create --base main

# Skip setup commands from .choir.yaml
choir env create --no-setup

# Override the default backend
choir env create --backend local
```

The create command:
1. Generates a unique environment ID (printed on success)
2. Creates a worktree at `~/.local/share/choir/worktrees/choir-<short-id>/`
3. Creates a new branch `env/<short-id>` from the base branch
4. Runs any setup commands defined in `.choir.yaml`

### env attach

Enter an existing environment's shell.

```bash
# Attach using the short ID (prefix matching works)
choir env attach a1b2
```

Use this to work in an environment's directory. When you exit the shell, the environment continues to exist.

### env list

Show all environments.

```bash
# List all active environments
choir env list

# Alias
choir env ls

# Include removed/failed environments
choir env list --all

# Filter to current repository only
choir env list --repo

# Filter by backend
choir env list --backend worktree
```

Example output:
```
ID        STATUS  BRANCH       CREATED
a1b2c3d4  ready   env/a1b2c3d4  2h ago
e5f6g7h8  ready   env/e5f6g7h8  5m ago
```

### env status

Show detailed information about an environment.

```bash
choir env status a1b2
```

Example output:
```
ID:          a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Short ID:    a1b2c3d4
Status:      ready
Backend:     local
Path:        /Users/me/.local/share/choir/worktrees/choir-a1b2c3d4
Branch:      env/a1b2c3d4
Base Branch: main
Repository:  /Users/me/projects/myrepo
Remote:      git@github.com:user/myrepo.git
Created:     2025-01-15 10:30:45
```

### env rm

Remove an environment and its worktree.

```bash
# Remove an environment (prompts for confirmation if ready)
choir env rm a1b2

# Force remove without confirmation
choir env rm -f a1b2
```

This destroys the worktree directory and removes the environment from the database. Any uncommitted changes in the worktree will be lost.

### init

Create a `.choir.yaml` configuration template.

```bash
# Create template in current directory
choir init

# Overwrite existing file
choir init --force
```

### config

View or modify global configuration.

```bash
# Show current configuration
choir config show

# Open configuration in $EDITOR
choir config edit
```

## Workflows

### Parallel Feature Development

Work on multiple features simultaneously without branch switching:

```bash
# Terminal 1: Create environment for authentication work
choir env create --base main
# Output: a1b2c3d4
choir env attach a1b2
# Now in ~/.local/share/choir/worktrees/choir-a1b2c3d4 on branch env/a1b2c3d4
# Make changes, commit as needed

# Terminal 2: Create environment for API refactor
choir env create --base main
# Output: e5f6g7h8
choir env attach e5f6
# Work independently from Terminal 1

# Check status from main repo
cd /path/to/main-repo
choir env list
```

### Bug Fix While Feature In Progress

Pause feature work to fix an urgent bug:

```bash
# You're working on a feature
choir env create --base main
# Output: abc12345
choir env attach abc1

# Urgent bug comes in - create another environment
# (In a new terminal)
choir env create --base main
# Output: def67890
choir env attach def6

# Fix the bug, commit, push
git add . && git commit -m "Fix crash"
git push origin env/def67890

# Remove the hotfix environment when done
choir env rm def6

# Continue feature work in original terminal
```

### Code Review Workflow

Review a PR while keeping your work intact:

```bash
# Your current work
choir env create
choir env attach <id>

# Review someone's PR (in new terminal)
choir env create --base feature/their-branch
# Output: rev12345
choir env attach rev1

# Examine their code, run tests
go test ./...

# Clean up when done reviewing
choir env rm rev1
```

### Experiment Safely

Try risky changes without affecting your main work:

```bash
# Create an experimental environment
choir env create --base main
# Output: exp98765
choir env attach exp9

# Try things out - if it doesn't work, just remove it
choir env rm exp9

# Nothing in your main repo was affected
```

## Tips

### Environment IDs

Environment IDs are auto-generated hex strings. You can use any unique prefix to reference them:

```bash
# Full ID: a1b2c3d4e5f6g7h8
choir env attach a1b2c3d4e5f6g7h8  # Full ID
choir env attach a1b2c3d4          # Short ID (8 chars)
choir env attach a1b2              # Prefix (if unique)
choir env attach a1                # Shorter prefix (if unique)
```

### Finding Your Environments

```bash
# From anywhere, list all environments
choir env list

# Get detailed info about a specific environment
choir env status a1b2
```

### Cleaning Up

Regularly clean up finished environments:

```bash
# See all environments including old ones
choir env list --all

# Remove environments you're done with
choir env rm a1b2
choir env rm e5f6
```

### Git Operations

Each environment has its own branch. Standard git operations work:

```bash
# Inside an environment worktree
git status
git add .
git commit -m "Progress on feature"
git push origin env/a1b2c3d4

# Create a PR from the environment branch
gh pr create --base main
```

## Configuration

### Project Configuration

Create a `.choir.yaml` in your repository root to configure environment setup:

```bash
choir init
```

Or create it manually:

```yaml
version: 1

# Commands to run after environment creation
# Working directory: repository root
setup:
  - npm install
  - docker compose up -d

# Environment variables
env:
  # Literal value
  NODE_ENV: development

  # Reference host environment variable
  DATABASE_URL: ${DATABASE_URL}

  # Read value from file
  API_KEY:
    from_file: ~/.secrets/api-key

# Files to copy into VM environments
files:
  - source: ~/.aws
    target: /home/ubuntu/.aws
    readonly: true

# Resource overrides (for VM backends)
resources:
  memory: 8GB
  cpus: 8

# Branch prefix (default: "env/")
branch_prefix: agent/
```

### Global Configuration

Global settings are stored at `~/.config/choir/config.yaml`:

```bash
# View current configuration
choir config show

# Edit configuration
choir config edit
```

Example global configuration:

```yaml
version: 1

default_backend: local

credentials:
  claude_config: ~/.claude
  ssh_keys: ~/.ssh
  git_config: ~/.gitconfig
  github_cli: ~/.config/gh

backends:
  local:
    type: lima
    cpus: 4
    memory: 4GB
    disk: 50GB
    vm_type: vz
```

## Troubleshooting

### "not in a git repository"

Run choir commands from within a git repository.

### "cannot create environment from detached HEAD"

You're on a detached HEAD. Specify a base branch:
```bash
choir env create --base main
```

### "environment not found"

The environment ID prefix doesn't match any environment:
```bash
# List all environments to find the correct ID
choir env list --all
```

### "ambiguous environment ID"

The prefix matches multiple environments. Use a longer prefix:
```bash
# If "a1" matches multiple environments
choir env attach a1b2  # Use more characters
```

### Environment shows "failed" status

The environment was created but setup didn't complete. Check what went wrong and try again:
```bash
# Get details about the failed environment
choir env status <id>

# Remove and recreate
choir env rm <id>
choir env create --base main
```
