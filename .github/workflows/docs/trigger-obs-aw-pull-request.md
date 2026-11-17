# Observability Agentic Workflow — Pull Request

Documents the Observability Agentic Workflow — Pull Request GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-obs-aw-pull-request.yml`](../trigger-obs-aw-pull-request.yml) |
| Trigger | `pull_request` |

## Purpose

Dispatches to the configured reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-pull-request.yml@main`. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: types: `["opened", "synchronize", "reopened", "labeled"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run-obs-aw-pull-request | No job-level condition or runner declared | None declared | reusable workflow `elastic/oblt-aw/.github/workflows/obs-aw-event-pull-request.yml@main` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; run-obs-aw-pull-request: `{"actions": "read", "contents": "write", "copilot-requests": "write", "discussions": "write", "id-token": "write", "issues": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-obs-aw-pull-request.yml`](../trigger-obs-aw-pull-request.yml)
