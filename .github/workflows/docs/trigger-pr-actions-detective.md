# PR Actions Detective

Documents the PR Actions Detective GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-pr-actions-detective.yml`](../trigger-pr-actions-detective.yml) |
| Trigger | `workflow_run` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-pr-actions-detective.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_run`: workflows: `["check-default", "pre-commit", "golangci-lint", "check-docs", "check-dev-tools"]`; types: `["completed"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | condition `github.event.workflow_run.conclusion == 'failure' && toJSON(github.event.workflow_run.pull_requests) != '[]'` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-pr-actions-detective.lock.yml@v0`; input names `["report-failure-as-issue", "setup-commands"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, setup-commands |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "read", "copilot-requests": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-pr-actions-detective.yml`](../trigger-pr-actions-detective.yml)
