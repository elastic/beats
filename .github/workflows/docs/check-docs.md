# check-docs

Documents the check-docs GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/check-docs.yml`](../check-docs.yml) |
| Trigger | `pull_request` |

## Purpose

Runs the configured `check` job(s) for **check-docs**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: paths: `[".github/workflows/check-docs.yml", "**/*.asciidoc", "**/*.md", "deploy/kubernetes/*-kubernetes.yaml"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| check | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; Update package lists runs a shell command; Install libpcap-dev runs a shell command; Install libsystemd-dev runs a shell command; Install librpm-dev runs a shell command; Fix Code is not compatible with Python 3.12 uses `actions/setup-python@v7`; Fix Code is not compatible with Python 3.12 input names `["python-version"]`; Run check runs a shell command |

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

- Workflow: [`.github/workflows/check-docs.yml`](../check-docs.yml)
