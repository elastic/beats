# github-commands-comment

Documents the github-commands-comment GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/github-commands-comment.yml`](../github-commands-comment.yml) |
| Trigger | `pull_request_target` |

## Purpose

Runs the configured `comment` job(s) for **github-commands-comment**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request_target`: types: `["opened"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| comment | runner `ubuntu-latest` | None declared | Step uses `elastic/oblt-actions/elastic/github-commands@v1` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No workflow or action inputs declared. |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; comment: `{"pull-requests": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/github-commands-comment.yml`](../github-commands-comment.yml)
