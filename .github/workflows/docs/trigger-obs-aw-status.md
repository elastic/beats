# Observability Agentic Workflow — Status

Documents the Observability Agentic Workflow — Status GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-obs-aw-status.yml`](../trigger-obs-aw-status.yml) |
| Trigger | `status` |

## Purpose

Dispatches to the configured reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-status.yml@main`. The checked-in source does not state a separate historical rationale.

## When it runs

- `status`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run-obs-aw-status | condition `github.event.state == 'failure' && contains(github.event.context, 'buildkite')` | None declared | reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-status.yml@main`; secret names `["BUILDKITE_API_TOKEN"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; run-obs-aw-status: `{"actions": "read", "contents": "read", "copilot-requests": "write", "id-token": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | `BUILDKITE_LOGS_API_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-obs-aw-status.yml`](../trigger-obs-aw-status.yml)
