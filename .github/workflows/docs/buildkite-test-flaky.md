# Buildkite Test Flaky

Documents the Buildkite Test Flaky GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/buildkite-test-flaky.yml`](../buildkite-test-flaky.yml) |
| Trigger | `schedule`, `workflow_dispatch` (scheduled) |

## Purpose

Runs the configured `run` job(s) for **Buildkite Test Flaky**. The checked-in source does not state a separate historical rationale.

## When it runs

- `schedule`: Runs on a schedule. See the [workflow configuration](../buildkite-test-flaky.yml) for current timing.
- `workflow_dispatch`.

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | runner `ubuntu-latest` | None declared | fetch-token uses `elastic/oblt-actions/github/create-token@v1`; Step uses `elastic/oblt-actions/buildkite/flaky-report@v1`; Step input names `["days", "github-issue-title-prefix", "github-label", "github-repo", "github-token", "max-issues", "org", "test-suite-id", "token"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | elastic/oblt-actions/buildkite/flaky-report@v1 input names: days, github-issue-title-prefix, github-label, github-repo, github-token, max-issues, org, test-suite-id, token |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}`; run: `{"id-token": "write"}` |
| Secrets | `BUILDKITE_TEST_ENGINE_TOKEN` |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/buildkite-test-flaky.yml`](../buildkite-test-flaky.yml)
