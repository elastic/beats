# Sweeper: OTel BeatReceiver Global State Isolation

Documents the Sweeper: OTel BeatReceiver Global State Isolation GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/sweep-otel-beatreceiver-isolation.yml`](../sweep-otel-beatreceiver-isolation.yml) |
| Trigger | `schedule`, `workflow_dispatch` (scheduled) |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-code-quality-audit.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `schedule`: Runs on a schedule. See the [workflow configuration](../sweep-otel-beatreceiver-isolation.yml) for current timing.
- `workflow_dispatch`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | No job-level condition or runner declared | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-code-quality-audit.lock.yml@v0`; input names `["report-failure-as-issue", "title-prefix", "severity-threshold", "additional-instructions"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, title-prefix, severity-threshold, additional-instructions |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "read", "copilot-requests": "write", "issues": "write", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/sweep-otel-beatreceiver-isolation.yml`](../sweep-otel-beatreceiver-isolation.yml)
