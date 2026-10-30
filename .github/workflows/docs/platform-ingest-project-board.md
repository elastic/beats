# Add issue to Ingest project

Documents the Add issue to Ingest project GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/platform-ingest-project-board.yml`](../platform-ingest-project-board.yml) |
| Trigger | `issues` |

## Purpose

Runs the configured `add_to_ingest_project` job(s) for **Add issue to Ingest project**. The checked-in source does not state a separate historical rationale.

## When it runs

- `issues`: types: `["labeled"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| add_to_ingest_project | runner `ubuntu-latest` | None declared | add_to_project uses `octokit/graphql-action@v2.x`; add_to_project input names `["query", "project_id", "content_id"]`; set_elastic_agent_area uses `octokit/graphql-action@v2.x`; set_elastic_agent_area input names `["query", "item_id", "project_id", "area_field_id", "area_id"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | octokit/graphql-action@v2.x input names: query, project_id, content_id; octokit/graphql-action@v2.x input names: query, item_id, project_id, area_field_id, area_id |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}` |
| Secrets | `PROJECT_ASSIGNER_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/platform-ingest-project-board.yml`](../platform-ingest-project-board.yml)
