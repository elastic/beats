# bump-elastic-stack-snapshot

Documents the bump-elastic-stack-snapshot GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/bump-elastic-stack-snapshot.yml`](../bump-elastic-stack-snapshot.yml) |
| Trigger | `workflow_dispatch`, `schedule` (scheduled) |

## Purpose

Runs the configured `filter`, `bump-elastic-stack` job(s) for **bump-elastic-stack-snapshot**. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `schedule`: Runs on a schedule. See the [workflow configuration](../bump-elastic-stack-snapshot.yml) for current timing.

## Execution flow

```mermaid
flowchart TD
  j1["filter"]
  j2["bump-elastic-stack"]
  j1 --> j2
```

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| filter | runner `ubuntu-latest` | None declared | generator uses `elastic/oblt-actions/elastic/active-branches@v1`; generator input names `["filter-branches"]`; outputs `["matrix"]` |
| bump-elastic-stack | runner `ubuntu-latest; matrix '${{ fromJson(needs.filter.outputs.matrix) }}'` | `["filter"]` | Step uses `actions/checkout@v7`; Step uses `elastic/oblt-actions/updatecli/run-and-notify@v1`; Step input names `["command", "slack-bot-token", "slack-channel-id", "slack-message", "slack-send-when", "version-file"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | elastic/oblt-actions/elastic/active-branches@v1 input names: filter-branches; elastic/oblt-actions/updatecli/run-and-notify@v1 input names: command, slack-bot-token, slack-channel-id, slack-message, slack-send-when, version-file |
| Outputs | filter: matrix |
| Permissions | workflow: `{"contents": "read"}`; bump-elastic-stack: `{"contents": "write", "pull-requests": "write"}` |
| Secrets | `GITHUB_TOKEN`, `SLACK_BOT_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/bump-elastic-stack-snapshot.yml`](../bump-elastic-stack-snapshot.yml)
- Local dependency: [`.github/workflows/updatecli.d/bump-elastic-stack-snapshot.yml`](../updatecli.d/bump-elastic-stack-snapshot.yml)
- Local dependency: [`.github/workflows/updatecli.d/values.d/scm.yml`](../updatecli.d/values.d/scm.yml)
