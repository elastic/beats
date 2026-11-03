# post-dependabot

Documents the post-dependabot GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/post-dependabot.yml`](../post-dependabot.yml) |
| Trigger | `push` |

## Purpose

Runs the configured `update-notice`, `manage-pr-reviewers` job(s) for **post-dependabot**. The checked-in source does not state a separate historical rationale.

## When it runs

- `push`: branches: `["dependabot/go_modules/**"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| update-notice | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file"]`; update NOTICE.txt runs a shell command; check for modified NOTICE.txt runs a shell command; commit NOTICE.txt runs a shell command |
| manage-pr-reviewers | runner `ubuntu-latest` | None declared | Add reviewers (obs-infraobs-integrations) if PR has Team:Obs-InfraObs label runs a shell command |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-go@v7 input names: go-version-file |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; update-notice: `{"contents": "write"}`; manage-pr-reviewers: `{"pull-requests": "write", "repository-projects": "read"}` |
| Secrets | `GITHUB_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/post-dependabot.yml`](../post-dependabot.yml)
