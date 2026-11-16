# Dra Prep Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/dra-prep-pipeline.yml`](../dra-prep-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/dra-prep-pipeline.yml`](../dra-prep-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **Dra Prep Pipeline**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| :package: DRA Prep (${WORKFLOW}) | agents `{"ephemeralStorage": "80Gi", "provider": "k8s"}` | None declared | runs `.buildkite/scripts/stage-dra-artifacts.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/dra-prep#v0.1.7`; artifacts `["artifacts/dra/beats/*/manifest-*.json"]`; key `dra-prep-${WORKFLOW}`; timeout 30 minutes |
| :memo: Annotate DRA build (${WORKFLOW}) | condition `build.env('DRY_RUN') != "true"`; agents `{"provider": "k8s"}` | `dra-prep-${WORKFLOW}` | runs `.buildkite/scripts/dra-annotate.sh`; plugins `elastic/oblt-google-auth#v1.3.1`; timeout 5 minutes |
| :pipeline: DRA processing for beats (${WORKFLOW}) | condition `build.env('DRY_RUN') != "true"` | `dra-prep-${WORKFLOW}` | triggers pipeline `unified-release-dra-processing`; trigger build fields `["env", "env.DRA_PRODUCT_ID", "env.DRA_STACK_VERSION", "env.DRA_WORKFLOW"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `DRA_PRODUCT_ID`, `DRA_STACK_VERSION`, `DRA_UPLOAD`, `DRA_WORKFLOW`, `STACK_VERSION`, `VERSION_QUALIFIER`, `WORKFLOW` |
| Outputs | `artifacts/dra/beats/*/manifest-*.json` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/dra-prep-pipeline.yml`](../dra-prep-pipeline.yml)
- Referenced local file: [`.buildkite/scripts/dra-annotate.sh`](../scripts/dra-annotate.sh)
- Referenced local file: [`.buildkite/scripts/packaging/resolve-dra-version.sh`](../scripts/packaging/resolve-dra-version.sh)
- Referenced local file: [`.buildkite/scripts/stage-dra-artifacts.sh`](../scripts/stage-dra-artifacts.sh)
- Referenced local file: [`.buildkite/scripts/version_qualifier.sh`](../scripts/version_qualifier.sh)
