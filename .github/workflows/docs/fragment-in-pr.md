# fragment-in-pr

Documents the fragment-in-pr GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/fragment-in-pr.yml`](../fragment-in-pr.yml) |
| Trigger | `pull_request` |

## Purpose

Runs the configured `fragments` job(s) for **fragment-in-pr**. The checked-in source does not state a separate historical rationale.

## When it runs

- `pull_request`: types: `["labeled", "unlabeled", "opened", "reopened", "synchronize"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| fragments | condition `!contains(github.event.pull_request.labels.*.name, 'skip-changelog') && !contains(github.event.pull_request.labels.*.name, 'backport')`; runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; check pr-has-fragment runs a shell command; get beats-agent version runs a shell command; check fragment-syntax runs a shell command |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-go@v7 input names: go-version-file |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/fragment-in-pr.yml`](../fragment-in-pr.yml)
