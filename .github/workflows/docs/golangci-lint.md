# golangci-lint

Documents the golangci-lint GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/golangci-lint.yml`](../golangci-lint.yml) |
| Trigger | `workflow_dispatch`, `pull_request` |

## Purpose

Runs the configured `golangci` job(s) for **golangci-lint**. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `pull_request`: paths: `["**/*.go", "**/go.mod", "**/go.sum", ".go-version", ".golangci.yml", ".github/workflows/golangci-lint.yml"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| golangci | runner `${{ matrix.os }}; matrix '{"os": ["ubuntu-latest", "macos-latest", "windows-latest"]}'` | None declared | Step uses `actions/checkout@v7`; Step input names `["fetch-depth"]`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; Install Apt Package runs a shell command; golangci-lint uses `golangci/golangci-lint-action@ba0d7d2ec06a0ea1cb5fa41b2e4a3ab91d21278a`; golangci-lint input names `["version-file", "args"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/checkout@v7 input names: fetch-depth; actions/setup-go@v7 input names: go-version-file; golangci-lint input names: version-file, args |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/golangci-lint.yml`](../golangci-lint.yml)
