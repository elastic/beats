# mergify backport labels copier

Documents the mergify backport labels copier GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/mergify-labels-copier.yml`](../mergify-labels-copier.yml) |
| Trigger | `pull_request` |

## Purpose

Runs the configured `mergify-backport-labels-copier` job(s) for **mergify backport labels copier**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: types: `["opened"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| mergify-backport-labels-copier | condition `startsWith(github.head_ref, 'mergify/bp/')`; runner `ubuntu-latest` | None declared | Step uses `elastic/oblt-actions/mergify/labels-copier@v1`; Step input names `["excluded-labels-regex"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | elastic/oblt-actions/mergify/labels-copier@v1 input names: excluded-labels-regex |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; mergify-backport-labels-copier: `{"pull-requests": "write", "repository-projects": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/mergify-labels-copier.yml`](../mergify-labels-copier.yml)
