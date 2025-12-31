---
name: writing-git-commit-messages-and-pull-requests
description: Instructions for how to write git commit messages and pull requests. Use when instructed to make a commit or make a PR.
---

## Writing commit messages

The body of the commit message should emphasize the WHY of the work being added in the commit. This is as important as the 'what'.

## Writing pull request messages

Like the body of a commit message, we should emphasize the WHY of the work being added.

If the pull request contains multiple commits, the pull request body should summarize the commits together. Otherwise, the pull request can be the same as the commit message.

## Commits and pull requests citing GitHub issues

If a pull request is being made to satisfy a specific github issue, that issue should be referenced in the pull request body. If the pull request CLOSES the github issue, that should also be cited with "Closes #<issue number>"

If, during the implementation of work to satisfy a ticket, a deviation was made away from the design or direction specified in the issue body, that deviation should be explicitly called out in the PR body.
