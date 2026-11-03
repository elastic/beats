# Backport to active branches

Documents the Backport to active branches GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/backport-active.yml`](../backport-active.yml) |
| Trigger | `pull_request_target` |

## Purpose

Runs the configured `backport` job(s) for **Backport to active branches**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request_target`: types: `["closed", "labeled"]`; branches: `["main"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| backport | condition `github.event.pull_request.merged == true && (   (github.event.action == 'closed' && contains(toJSON(github.event.pull_request.labels.*.name), 'backport-active-')) \|\|   (github.event.action == 'labeled' && startsWith(github.event.label.name, 'backport-active-')) )`; runner `ubuntu-latest` | None declared | Step uses `elastic/oblt-actions/github/backport-active@v1` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read", "pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/backport-active.yml`](../backport-active.yml)
