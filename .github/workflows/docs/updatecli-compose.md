# updatecli-compose

Documents the updatecli-compose GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/updatecli-compose.yml`](../updatecli-compose.yml) |
| Trigger | `workflow_dispatch`, `schedule` (scheduled) |

## Purpose

Runs the configured `compose` job(s) for **updatecli-compose**. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `schedule`: Runs on a schedule. See the [workflow configuration](../updatecli-compose.yml) for current timing.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| compose | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `docker/login-action@dbcb813823bdd20940b903addbd779551569679f`; Step input names `["registry", "username", "password"]`; Step uses `elastic/oblt-actions/updatecli/run-and-notify@v1`; Step input names `["command", "slack-bot-token", "slack-channel-id", "slack-message", "slack-send-when", "version-file"]`; Step uses `elastic/oblt-actions/updatecli/run-and-notify@v1`; Step input names `["command", "slack-bot-token", "slack-channel-id", "slack-message", "slack-send-when", "version-file"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | docker/login-action@dbcb813823bdd20940b903addbd779551569679f input names: registry, username, password; elastic/oblt-actions/updatecli/run-and-notify@v1 input names: command, slack-bot-token, slack-channel-id, slack-message, slack-send-when, version-file; elastic/oblt-actions/updatecli/run-and-notify@v1 input names: command, slack-bot-token, slack-channel-id, slack-message, slack-send-when, version-file |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; compose: `{"contents": "write", "packages": "read", "pull-requests": "write"}` |
| Secrets | `GITHUB_TOKEN`, `SLACK_BOT_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/updatecli-compose.yml`](../updatecli-compose.yml)
