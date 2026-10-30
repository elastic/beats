# pre-commit

Documents the pre-commit GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/pre-commit.yml`](../pre-commit.yml) |
| Trigger | `pull_request`, `push` |

## Purpose

Runs the configured `pre-commit` job(s) for **pre-commit**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`.
- `push`: branches: `["main", "7.1*", "8.*", "9.*"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| pre-commit | runner `ubuntu-latest` | None declared | Step uses `elastic/oblt-actions/pre-commit@v1` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/pre-commit.yml`](../pre-commit.yml)
