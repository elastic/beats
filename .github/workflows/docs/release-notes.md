# Generate release notes

Documents the Generate release notes GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/release-notes.yml`](../release-notes.yml) |
| Trigger | `workflow_dispatch` |

## Purpose

Dispatches to the configured reusable workflow `elastic/elastic-agent-changelog-tool/.github/workflows/generate-release-notes.yml@main`. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`: inputs: `{"bc_commit_sha": {"description": "Commit SHA for the build candidate", "required": true, "type": "string"}, "version": {"description": "Version to be released (for example, '9.2.1')", "required": true, "type": "string"}}`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| build-release-notes | No job-level condition or runner declared | None declared | reusable workflow `elastic/elastic-agent-changelog-tool/.github/workflows/generate-release-notes.yml@main`; input names `["product", "labels", "bc_commit_sha", "version", "release_manager", "branching_model"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | workflow_dispatch.bc_commit_sha (required: True; type: string; Commit SHA for the build candidate); workflow_dispatch.version (required: True; type: string; Version to be released (for example, `9.2.1`)); build-release-notes input names: product, labels, bc_commit_sha, version, release_manager, branching_model |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; build-release-notes: `{"contents": "read", "id-token": "write"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/release-notes.yml`](../release-notes.yml)
