# Observability Agentic Workflow — Schedule (daily)

Documents the Observability Agentic Workflow — Schedule (daily) GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-obs-aw-schedule-daily.yml`](../trigger-obs-aw-schedule-daily.yml) |
| Trigger | `schedule`, `workflow_dispatch` (scheduled) |

## Purpose

Dispatches to the configured reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-schedule.yml@main`. The checked-in source does not state a separate historical rationale.

## When it runs

- `schedule`: Runs on a schedule. See the [workflow configuration](../trigger-obs-aw-schedule-daily.yml) for current timing.
- `workflow_dispatch`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run-obs-aw-schedule-daily | No job-level condition or runner declared | None declared | reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-schedule.yml@main`; input names `["schedule-profile"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run-obs-aw-schedule-daily input names: schedule-profile |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; run-obs-aw-schedule-daily: `{"actions": "read", "contents": "write", "copilot-requests": "write", "id-token": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-obs-aw-schedule-daily.yml`](../trigger-obs-aw-schedule-daily.yml)
