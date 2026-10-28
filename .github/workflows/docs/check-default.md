# check-default

Documents the check-default GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/check-default.yml`](../check-default.yml) |
| Trigger | `pull_request`, `push` |

## Purpose

Runs the configured `check` job(s) for **check-default**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`.
- `push`: branches: `["main", "7.1*", "8.*"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| check | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; Fix Code is not compatible with Python 3.12 uses `actions/setup-python@v7`; Fix Code is not compatible with Python 3.12 input names `["python-version"]`; Run check-default runs a shell command |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-go@v7 input names: go-version-file; Fix Code is not compatible with Python 3.12 input names: python-version |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/check-default.yml`](../check-default.yml)
