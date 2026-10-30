# catalog-info

Documents the catalog-info GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/catalog-info.yml`](../catalog-info.yml) |
| Trigger | `pull_request` |

## Purpose

Runs the configured `validate` job(s) for **catalog-info**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: branches: `["main"]`; paths: `["catalog-info.yaml"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| validate | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `elastic/oblt-actions/elastic/validate-catalog@v1` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; validate: `{"contents": "read", "packages": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/catalog-info.yml`](../catalog-info.yml)
