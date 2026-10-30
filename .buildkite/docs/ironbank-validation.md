# Ironbank Validation

Documents the Buildkite pipeline configuration in [`.buildkite/ironbank-validation.yml`](../ironbank-validation.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/ironbank-validation.yml`](../ironbank-validation.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **Ironbank Validation**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| :package: :test_tube: Package and Validate Iron Bank for x-pack/heartbeat | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "n2-standard-8", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/heartbeat/build/distributions/*", "x-pack/heartbeat/build/heartbeat-ironbank-*-docker-build-context/*"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `BEAT_VERSION`, `IMAGE_UBUNTU_X86_64` |
| Outputs | `x-pack/heartbeat/build/distributions/*`, `x-pack/heartbeat/build/heartbeat-ironbank-*-docker-build-context/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/ironbank-validation.yml`](../ironbank-validation.yml)
