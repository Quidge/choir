---
name: github-issue-workflow
description: Manage GitHub issue workflow - start work on issues, pause, complete with PR links, and find available issues. Use when the user wishes to begin work on a known issue, or wants to create an issue for work.
---

# Issue Workflow Manager

Automatically manage labels, assignments, comments, branches, and PR links on GitHub issues.

## When to Use This Skill

**Start work:**
- "I want to work on issue #X"
- "Let me pick up issue #X"
- "Taking issue #X"
- "Let's work on issue #X"

**Pause work:**
- "Pausing work on #X"
- "Stopping work on #X"
- "Put issue #X back"

**Complete work:**
- "I'm done with issue #X"
- "Finished #X, created PR #Y"
- "Complete issue #X"

**Find issues:**
- "What issues are available?"
- "Show me open issues"

## Instructions

### Starting Work on an Issue

1. **Get the GitHub username:**
   ```bash
   gh api user --jq '.login'
   ```

2. **Check if issue is already in-progress:**
   ```bash
   gh issue view <issue-number> --json labels,assignees,state --jq '{labels: [.labels[].name], assignees: [.assignees[].login], state: .state}'
   ```
   - If state is not "OPEN", inform user the issue is closed
   - If labels contain "in-progress" AND assignees include someone other than current user:
     - **WARN the user**: "Issue #X is currently being worked on by @Y. Do you want to take it over?"
     - Wait for confirmation before proceeding

3. **Ensure labels exist (create if needed):**
   ```bash
   gh label create "in-progress" --description "Work is actively in progress" --color "FFA500" --force
   gh label create "assigned:<username>" --description "Assigned to <username>" --color "0E8A16" --force
   gh label create "done" --description "Work completed" --color "0E8A16" --force
   ```

4. **Add labels and assign user:**
   ```bash
   gh issue edit <issue-number> --add-label "in-progress" --add-label "assigned:<username>" --add-assignee "<username>"
   ```

5. **Add a comment:**
   ```bash
   gh issue comment <issue-number> --body "🚀 @<username> has started working on this issue."
   ```

6. **Create feature branch from main:**
   ```bash
   git fetch origin
   git branch -r | grep -E "(issue-<number>|feature/issue-<number>|<number>-)" || echo "no branch"
   ```
   - If no branch exists:
     ```bash
     git checkout main && git pull origin main
     git checkout -b issue-<issue-number>
     git push -u origin issue-<issue-number>
     ```
   - If branch exists, ask user if they want to check it out

7. **Create task with TodoWrite:**
   - Content: "Work on: <issue-title> (#<number>)"
   - Status: "in_progress"

8. **Confirm to user:** "Started work on issue #X - assigned you, added labels, commented, and created branch issue-X from main"

### Pausing Work on an Issue

Use when user wants to stop temporarily but may return later.

1. **Get the GitHub username:**
   ```bash
   gh api user --jq '.login'
   ```

2. **Remove workflow labels (keep assignment):**
   ```bash
   gh issue edit <issue-number> --remove-label "in-progress" --remove-label "assigned:<username>"
   ```

3. **Generate work summary** by reviewing:
   - Recent commits on the current branch
   - Files changed
   - Any TODOs or incomplete items

4. **Add summary comment:**
   ```bash
   gh issue comment <issue-number> --body "⏸️ @<username> has paused work on this issue.

   **Summary of work done:**
   - [List key changes/commits]
   - [Files modified]
   - [Remaining TODOs if applicable]"
   ```

5. **Update TodoWrite:** Mark task as "pending" (paused, not completed)

6. **Confirm to user:** "Paused work on issue #X - removed in-progress labels and added summary comment"

### Completing Work on an Issue

Use when work is finished and ready to close.

1. **Get the GitHub username:**
   ```bash
   gh api user --jq '.login'
   ```

2. **Find related PRs:**
   ```bash
   gh pr list --search "head:issue-<number>" --state all --json number,url
   ```
   Or if PR number is known:
   ```bash
   gh pr view <pr-number> --json number,url
   ```

3. **Update labels:**
   ```bash
   gh issue edit <issue-number> --remove-label "in-progress" --remove-label "assigned:<username>" --add-label "done"
   ```

4. **Generate work summary** by reviewing:
   - All commits on the branch
   - Files changed
   - PRs created

5. **Add completion comment with PR links:**
   ```bash
   gh issue comment <issue-number> --body "✅ @<username> has completed work on this issue.

   **Summary of work done:**
   - [List key changes/commits]
   - [Files modified]

   **Related PR(s):** #<pr1>, #<pr2>"
   ```

6. **Update TodoWrite:** Mark task as "completed"

7. **Confirm to user:** "Completed issue #X - added 'done' label, linked PR(s), and posted summary"

### Finding Available Issues

```bash
gh issue list --assignee "" --state open --limit 10
```

Present the list to user with issue numbers and titles.

## Example Workflows

### Starting Work
```
User: "I want to work on issue #42"

1. Get username: gh api user --jq '.login' → "myuser"
2. Check issue: gh issue view 42 --json labels,assignees,state
3. If unassigned and open:
   - Create labels if needed
   - Assign: gh issue edit 42 --add-label "in-progress" --add-label "assigned:myuser" --add-assignee "myuser"
   - Comment: gh issue comment 42 --body "🚀 @myuser has started working on this issue."
   - Create branch from main
   - Create TodoWrite task
   - Confirm: "Started work on issue #42 - assigned you, created branch issue-42"
4. If already in-progress by another user:
   - "Issue #42 is currently being worked on by @otheruser. Do you want to take it over?"
```

### Completing Work
```
User: "I'm done with issue #42, created PR #15"

1. Get username
2. Find PRs: gh pr list --search "head:issue-42" or use #15
3. Update labels: gh issue edit 42 --remove-label "in-progress" --remove-label "assigned:myuser" --add-label "done"
4. Add comment with summary and PR link
5. Mark TodoWrite task as completed
6. Confirm: "Completed issue #42 - added 'done' label and linked to PR #15"
```

## Notes

- Labels are created automatically if they don't exist
- Feature branches follow pattern: `issue-<number>` and branch from `main`
- User is automatically assigned when starting work
- Comments track work history with timestamps
- Warn before taking over issues already in-progress by others
- **Pause** = stopping temporarily (keeps assignee, removes in-progress)
- **Complete** = work is done (adds "done" label, links PRs)
- Always link related PRs when completing an issue
- Requires GitHub CLI (`gh`) installed and authenticated
