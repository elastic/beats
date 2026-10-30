# Version Bump Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/version-bump-pipeline.yml`](../version-bump-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/version-bump-pipeline.yml`](../version-bump-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **Version Bump Pipeline**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| :rocket: Trigger ingest-dev minor/major release | condition `build.env("WORKFLOW") == "minor" \|\| build.env("WORKFLOW") == "major"` | None declared | triggers pipeline `ingest-dev-pipeline-release-minor-major`; trigger build fields `["branch", "message", "env", "env.REPOSITORY", "env.RELEASE_TYPE", "env.CURRENT_RELEASE", "env.DISABLE_DEV_ISSUE", "env.REVIEWERS"]`; key `bump-version-minor-major` |
| :rocket: Trigger ingest-dev patch release | condition `build.env("WORKFLOW") == "patch"` | None declared | triggers pipeline `ingest-dev-pipeline-release-patch`; trigger build fields `["branch", "message", "env", "env.REPOSITORY", "env.CURRENT_RELEASE", "env.DISABLE_DEV_ISSUE", "env.REVIEWERS"]`; key `bump-version-patch` |
| Ready to fetch for DRA artifacts? | No step-level condition or agent declared | None declared | key `block-get-dra-artifacts` |
| Fetch DRA Artifacts | agents `{"cpu": "250m", "ephemeralStorage": "1Gi", "image": "docker.elastic.co/release-eng/wolfi-build-essential-release-eng:latest", "memory": "512Mi"}` | `block-get-dra-artifacts` | plugins `elastic/version-bump-dra#v1.0.0`; key `fetch-dra-artifacts`; retry policy `{"automatic": [{"exit_status": "*", "limit": 2}], "manual": {"permit_on_passed": true}}`; timeout 240 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `BRANCH`, `BUILDKITE_BUILD_URL`, `CURRENT_RELEASE`, `DISABLE_DEV_ISSUE`, `NEW_VERSION`, `RELEASE_TYPE`, `REPO`, `REPOSITORY`, `REVIEWERS`, `WORKFLOW` |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/version-bump-pipeline.yml`](../version-bump-pipeline.yml)
