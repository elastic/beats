# xpack-heartbeat-macos-unit-tests

Documents the xpack-heartbeat-macos-unit-tests GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/xpack-heartbeat-macos-unit-tests.yml`](../xpack-heartbeat-macos-unit-tests.yml) |
| Trigger | `workflow_dispatch`, `pull_request`, `push` |

## Purpose

Builds and runs unit tests for **x-pack heartbeat** on the configured macOS runner using the shared unit-test action. The checked-in source does not state a separate historical rationale.

## When it runs

- `workflow_dispatch`.
- `pull_request`: paths: `["heartbeat/**", "x-pack/heartbeat/**", "go.mod", "dev-tools/**", "libbeat/**", "x-pack/libbeat/**", ".github/workflows/xpack-heartbeat-macos-unit-tests.yml", ".github/actions/unit-test/action.yml"]`
- `push`: branches: `["main"]`; paths: `["heartbeat/**", "x-pack/heartbeat/**", "go.mod", "dev-tools/**", "libbeat/**", "x-pack/libbeat/**", ".github/workflows/xpack-heartbeat-macos-unit-tests.yml", ".github/actions/unit-test/action.yml"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| unit-tests | runner `${{ matrix.os }}; matrix '{"os": ["macos-15-intel", "macos-15"]}'` | None declared | Step uses `actions/checkout@v7`; Step uses `actions/setup-node@v7`; Step input names `["node-version"]`; Step runs a shell command; Step uses `./.github/actions/unit-test`; Step input names `["beat"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | actions/setup-node@v7 input names: node-version; ./.github/actions/unit-test input names: beat |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"contents": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/xpack-heartbeat-macos-unit-tests.yml`](../xpack-heartbeat-macos-unit-tests.yml)
- Local dependency: [`.github/actions/unit-test/action.yml`](../../actions/unit-test/action.yml)
