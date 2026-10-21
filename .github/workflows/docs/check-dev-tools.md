# check-dev-tools

Documents the check-dev-tools GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/check-dev-tools.yml`](../check-dev-tools.yml) |
| Trigger | `pull_request` |

## Purpose

Runs the configured `check` job(s) for **check-dev-tools**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: paths: `[".github/workflows/check-dev-tools.yml", "dev-tools/**", "libbeat/scripts/Makefile"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| check | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; Step uses `actions/setup-python@v7`; Step input names `["cache", "python-version-file"]`; Run check/update runs a shell command |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-go@v7 input names: go-version-file; actions/setup-python@v7 input names: cache, python-version-file |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/check-dev-tools.yml`](../check-dev-tools.yml)
