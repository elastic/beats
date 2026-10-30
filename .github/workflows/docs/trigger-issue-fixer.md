# Issue Fixer

Documents the Issue Fixer GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-issue-fixer.yml`](../trigger-issue-fixer.yml) |
| Trigger | `issues` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-issue-fixer.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `issues`: types: `["labeled"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| flaky-test-fixer | condition `contains(github.event.issue.labels.*.name, 'flaky-test') && contains(github.event.issue.labels.*.name, 'triaged')` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-issue-fixer.lock.yml@v0`; input names `["report-failure-as-issue", "allowed-bot-users", "additional-instructions"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | flaky-test-fixer input names: report-failure-as-issue, allowed-bot-users, additional-instructions |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "write", "copilot-requests": "write", "discussions": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-issue-fixer.yml`](../trigger-issue-fixer.yml)
