# CLI Usage Guide

This guide covers common workflows for using choir to manage parallel agent workspaces.

## Quick Start

```bash
# Create a new agent and start working
choir spawn feature-auth

# In another terminal, create a second agent for a different task
choir spawn bugfix-login

# See all running agents
choir list

# When done, remove an agent
choir rm feature-auth
```

## Commands

### spawn

Create a new agent workspace with an isolated git worktree.

```bash
# Basic usage - creates agent on current branch
choir spawn my-task

# Spawn from a specific branch
choir spawn my-task --base main

# Skip setup commands from .choir.yaml
choir spawn my-task --no-setup
```

The spawn command:
1. Creates a worktree at `<repo-parent>/choir-<task-id>/`
2. Creates a new branch `agent/<task-id>` from the base branch
3. Runs any setup commands defined in `.choir.yaml`
4. Drops you into an interactive shell in the worktree

### attach

Reconnect to an existing agent's shell.

```bash
# Attach to an agent you created earlier
choir attach my-task
```

Use this when you've exited an agent's shell and want to continue working in that workspace.

### list

Show all agents.

```bash
# List all active agents
choir list

# Include removed/failed agents
choir list --all

# Filter to current repository only
choir list --repo

# Filter by backend
choir list --backend worktree
```

Example output:
```
TASK ID       STATUS    BRANCH              PATH
feature-auth  running   agent/feature-auth  /Users/me/choir-feature-auth
bugfix-login  running   agent/bugfix-login  /Users/me/choir-bugfix-login
```

### rm

Remove an agent and its worktree.

```bash
# Remove an agent
choir rm my-task
```

This destroys the worktree directory and removes the agent from the database. Any uncommitted changes in the worktree will be lost.

## Workflows

### Parallel Feature Development

Work on multiple features simultaneously without branch switching:

```bash
# Terminal 1: Work on authentication
choir spawn auth-feature --base main
# Now in /path/to/choir-auth-feature on branch agent/auth-feature
# Make changes, commit as needed

# Terminal 2: Work on API refactor
choir spawn api-refactor --base main
# Now in /path/to/choir-api-refactor on branch agent/api-refactor
# Work independently from Terminal 1

# Check status from main repo
cd /path/to/main-repo
choir list
```

### Bug Fix While Feature In Progress

Pause feature work to fix an urgent bug:

```bash
# You're working on a feature
choir spawn new-dashboard --base main

# Urgent bug comes in - create another agent
# (In a new terminal)
choir spawn hotfix-crash --base main

# Fix the bug, commit, push
git add . && git commit -m "Fix crash"
git push origin agent/hotfix-crash

# Remove the hotfix agent when done
choir rm hotfix-crash

# Continue feature work in original terminal
```

### Code Review Workflow

Review a PR while keeping your work intact:

```bash
# Your current work
choir spawn my-feature

# Review someone's PR (in new terminal)
choir spawn review-pr-123 --base feature/their-branch

# Examine their code, run tests
go test ./...

# Clean up when done reviewing
choir rm review-pr-123
```

### Experiment Safely

Try risky changes without affecting your main work:

```bash
# Create an experimental branch
choir spawn experiment-new-arch --base main

# Try things out - if it doesn't work, just remove it
choir rm experiment-new-arch

# Nothing in your main repo was affected
```

## Tips

### Naming Conventions

Use descriptive task IDs that indicate the work:
- `feature-auth` - Feature work
- `bugfix-login` - Bug fixes
- `refactor-api` - Refactoring
- `experiment-cache` - Experiments
- `review-pr-42` - Code reviews

### Finding Your Agents

```bash
# From anywhere, list all agents
choir list

# The PATH column shows where each worktree lives
```

### Cleaning Up

Regularly clean up finished agents:

```bash
# See all agents including old ones
choir list --all

# Remove agents you're done with
choir rm old-task-1
choir rm old-task-2
```

### Git Operations

Each agent has its own branch. Standard git operations work:

```bash
# Inside an agent worktree
git status
git add .
git commit -m "Progress on feature"
git push origin agent/my-task

# Create a PR from the agent branch
gh pr create --base main
```

## Configuration

Create a `.choir.yaml` in your repository root to configure agent setup:

```yaml
version: 1

# Commands to run when spawning a new agent
setup:
  - npm install
  - cp .env.example .env

# Environment variables for agents
env:
  NODE_ENV: development

# Branch prefix (default: "agent/")
branch_prefix: "agent/"
```

## Troubleshooting

### "agent already exists"

An agent with that task ID exists. Either:
- Use a different task ID
- Remove the existing agent: `choir rm <task-id>`

### "not in a git repository"

Run choir commands from within a git repository.

### "cannot spawn from detached HEAD"

You're on a detached HEAD. Specify a base branch:
```bash
choir spawn my-task --base main
```

### Agent shows "(pending)" path

The agent was created but provisioning didn't complete. Remove it and try again:
```bash
choir rm incomplete-agent
choir spawn new-agent
```
