# Docs PR Review

Documents the Docs PR Review GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-docs-pr-review.yml`](../trigger-docs-pr-review.yml) |
| Trigger | `issue_comment` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-estc-docs-pr-review.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `issue_comment`: types: `["created"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | condition `github.event.issue.pull_request != null && startsWith(github.event.comment.body, '/docs-review')` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-estc-docs-pr-review.lock.yml@v0`; input names `["report-failure-as-issue", "setup-commands"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, setup-commands |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "checks": "read", "contents": "read", "copilot-requests": "write", "issues": "read", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-docs-pr-review.yml`](../trigger-docs-pr-review.yml)
