# PR Review

Documents the PR Review GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-pr-review.yml`](../trigger-pr-review.yml) |
| Trigger | `pull_request` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-pr-review.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: types: `["opened", "synchronize", "reopened", "ready_for_review", "labeled", "unlabeled"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | condition `github.event.pull_request.draft == false && !contains(github.event.pull_request.labels.*.name, 'skip-auto-pr-review') && contains(fromJSON('["OWNER","MEMBER","COLLABORATOR"]'), github.event.pull_request.author_association)` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-pr-review.lock.yml@v0`; input names `["report-failure-as-issue", "setup-commands"]` |

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

- Workflow: [`.github/workflows/trigger-pr-review.yml`](../trigger-pr-review.yml)
