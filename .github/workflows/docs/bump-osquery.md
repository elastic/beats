# bump-osquery

Documents the bump-osquery GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/bump-osquery.yml`](../bump-osquery.yml) |
| Trigger | `workflow_dispatch`, `schedule` (scheduled) |

## Purpose

Runs the configured `bump` job(s) for **bump-osquery**. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `schedule`: Runs on a schedule. See the [workflow configuration](../bump-osquery.yml) for current timing.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| bump | runner `ubuntu-latest` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-go@v7`; Step input names `["go-version-file", "cache"]`; Step uses `elastic/oblt-actions/updatecli/run@v1`; Step input names `["command", "version-file"]`; Step uses `elastic/oblt-actions/slack/send@v1`; Step input names `["bot-token", "channel-id", "message"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-go@v7 input names: go-version-file, cache; elastic/oblt-actions/updatecli/run@v1 input names: command, version-file; elastic/oblt-actions/slack/send@v1 input names: bot-token, channel-id, message |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; bump: `{"contents": "write", "pull-requests": "write"}` |
| Secrets | `GITHUB_TOKEN`, `SLACK_BOT_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/bump-osquery.yml`](../bump-osquery.yml)
- Local dependency: [`.github/workflows/updatecli.d/bump-osquery.yml`](../updatecli.d/bump-osquery.yml)
- Local dependency: [`.github/workflows/updatecli.d/values.d/scm.yml`](../updatecli.d/values.d/scm.yml)
