# Mention in Issue

Documents the Mention in Issue GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-mention-in-issue.yml`](../trigger-mention-in-issue.yml) |
| Trigger | `issue_comment` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-mention-in-issue.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `issue_comment`: types: `["created"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | condition `github.event.issue.pull_request == null && (startsWith(github.event.comment.body, '/ai') \|\| contains(github.event.comment.body, '@claude'))` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-mention-in-issue.lock.yml@v0`; input names `["report-failure-as-issue", "setup-commands", "additional-instructions"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, setup-commands, additional-instructions |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "write", "copilot-requests": "write", "discussions": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-mention-in-issue.yml`](../trigger-mention-in-issue.yml)
