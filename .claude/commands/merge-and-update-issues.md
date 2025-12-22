---
argument-hint: [PR-number]
description: Merge a PR and update any related GitHub issues.
---

Use the GH cli (`gh`) to view the GitHub PR #$1. Keep in mind it may already be merged (or closed without merge).

Your goal is to determine which GitHub issues this PR related to (if any), and to ensure the related GH issue is marked as done (and closed) if suitable.

On ocassion a GitHub issue may need multiple PRs to implement the work and changes required, in which case the closure of a single PR may not necessitate the closing of the GH issue. That is an unusual scenario however.

Additionally, it's possible that the related GH issue is a child issue related to a larger issue. The larger issue should be viewed and checked if the body of that larger issue needs to be updated for accuracy.

## In case of a deviation

If the PR deviated from the written design in the directly related GH issue or any parent + sibling issues, in ways that would affect those issues, those issue bodies need to be updated to note the new changes.
