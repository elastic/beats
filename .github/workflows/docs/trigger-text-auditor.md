# Text Auditor

Documents the Text Auditor GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-text-auditor.yml`](../trigger-text-auditor.yml) |
| Trigger | `schedule`, `workflow_dispatch` (scheduled) |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-text-auditor.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `schedule`: Runs on a schedule. See the [workflow configuration](../trigger-text-auditor.yml) for current timing.
- `workflow_dispatch`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | No job-level condition or runner declared | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-text-auditor.lock.yml@v0`; input names `["report-failure-as-issue", "setup-commands"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, setup-commands |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "read", "copilot-requests": "write", "issues": "write", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-text-auditor.yml`](../trigger-text-auditor.yml)
