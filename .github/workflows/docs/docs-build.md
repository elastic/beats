# docs-build

Documents the docs-build GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/docs-build.yml`](../docs-build.yml) |
| Trigger | `pull_request`, `push`, `merge_group` |

## Purpose

Dispatches to the configured reusable workflow `elastic/docs-actions/.github/workflows/docs-build.yml@v1`. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: types: `["opened", "synchronize", "reopened"]`
- `push`: branches: `["main", "[0-9]+.[0-9]+"]`
- `merge_group`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| build | No job-level condition or runner declared | None declared | reusable workflow `elastic/docs-actions/.github/workflows/docs-build.yml@v1`; input names `["enable-vale-linting", "use-release-branches"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | build input names: enable-vale-linting, use-release-branches |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/docs-build.yml`](../docs-build.yml)
