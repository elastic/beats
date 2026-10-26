# docs-preview-cleanup

Documents the docs-preview-cleanup GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/docs-preview-cleanup.yml`](../docs-preview-cleanup.yml) |
| Trigger | `pull_request_target` |

## Purpose

Dispatches to the configured reusable workflow `elastic/docs-actions/.github/workflows/docs-preview-cleanup.yml@v1`. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request_target`: types: `["closed"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| cleanup | No job-level condition or runner declared | None declared | reusable workflow `elastic/docs-actions/.github/workflows/docs-preview-cleanup.yml@v1` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "none", "deployments": "write", "id-token": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/docs-preview-cleanup.yml`](../docs-preview-cleanup.yml)
