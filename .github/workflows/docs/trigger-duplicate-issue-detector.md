# Duplicate Issue Detector

Documents the Duplicate Issue Detector GitHub Actions workflow and its configured execution.

## At a glance

| | |
| --- | --- |
| Platform | GitHub Actions |
| Source | [`.github/workflows/trigger-duplicate-issue-detector.yml`](../trigger-duplicate-issue-detector.yml) |
| Trigger | `issues` |

## Purpose

Dispatches to the configured reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-duplicate-issue-detector.lock.yml@v0`. The checked-in source does not state a separate historical rationale.

## When it runs

- `issues`: types: `["opened"]`

## Execution flow

| Job | Runs when / runner | Depends on | Configured work |
| --- | --- | --- | --- |
| run | condition `contains(fromJSON('["OWNER","MEMBER","COLLABORATOR"]'), github.event.issue.author_association)` | None declared | reusable workflow `elastic/ai-github-actions/.github/workflows/gh-aw-duplicate-issue-detector.lock.yml@v0`; input names `["report-failure-as-issue", "detect-related-issues"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | run input names: report-failure-as-issue, detect-related-issues |
| Outputs | No job outputs, artifact upload, or cache action declared in this workflow. |
| Permissions | workflow: `{"actions": "read", "contents": "read", "copilot-requests": "write", "discussions": "write", "issues": "write", "pull-requests": "read"}` |
| Secrets | No secret references found in this workflow file. |

## Operational notes

Local composite actions and shell scripts used by this workflow are linked under Sources. External actions are identified by their `uses` reference in the job table; their implementation is maintained outside this repository.

## Sources

- Workflow: [`.github/workflows/trigger-duplicate-issue-detector.yml`](../trigger-duplicate-issue-detector.yml)
