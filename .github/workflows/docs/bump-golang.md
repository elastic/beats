# bump-golang

Documents the bump-golang GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/bump-golang.yml`](../bump-golang.yml) |
| Trigger | `workflow_dispatch`, `schedule` (scheduled) |

## Purpose

Runs the configured `bump` job(s) for **bump-golang**. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `schedule`: Runs on a schedule. See the [workflow configuration](../bump-golang.yml) for current timing.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| bump | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `elastic/oblt-actions/updatecli/run-and-notify@v1`; Step input names `["command", "slack-bot-token", "slack-channel-id", "slack-message", "slack-send-when", "version-file"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | elastic/oblt-actions/updatecli/run-and-notify@v1 input names: command, slack-bot-token, slack-channel-id, slack-message, slack-send-when, version-file |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; bump: `{"contents": "write", "pull-requests": "write"}` |
| Secrets | `GITHUB_TOKEN`, `SLACK_BOT_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/bump-golang.yml`](../bump-golang.yml)
- Local dependency: [`.github/workflows/updatecli.d/bump-golang.yml`](../updatecli.d/bump-golang.yml)
- Local dependency: [`.github/workflows/updatecli.d/values.d/scm.yml`](../updatecli.d/values.d/scm.yml)
