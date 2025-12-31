---
argument-hint: [pr-number]
description: React to feedback to a pull request.
allowed-tools: Read, Edit, Write, Grep, Glob, Bash(gh:*), Bash(git:*)
---

Feedback has been given on PR #$1 (possibly more than one comment or more than one reviewer). Review and analyze this feedback, consolidate it together into a uniquely numbered list (multiple reviews may be speaking to the same thing, so consolidate that), present that list, and present your own analysis for which items from the list you think you need to follow up on (if any).

After instruction from the user, if you end up following up on any of the items, create a new commit for those changes and cite the original feedback comment/review link(s) in your commit message. Then push the commit up to the PR branch.

Use notes from the writing-git-commit-messages-and-pull-requests skill for instruction on how to write the commit message. 
