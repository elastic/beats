# docs-deploy

Documents the docs-deploy GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/docs-deploy.yml`](../docs-deploy.yml) |
| Trigger | `workflow_run` |

## Purpose

Dispatches to the configured reusable workflow `elastic/docs-actions/.github/workflows/docs-deploy.yml@v1`. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_run`: workflows: `["docs-build"]`; types: `["completed"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| deploy | No job-level condition or runner declared | None declared | reusable workflow `elastic/docs-actions/.github/workflows/docs-deploy.yml@v1`; input names `["enable-vale-linting", "use-release-branches"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | deploy input names: enable-vale-linting, use-release-branches |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "read", "deployments": "write", "id-token": "write", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/docs-deploy.yml`](../docs-deploy.yml)
