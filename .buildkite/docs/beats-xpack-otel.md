# beats-xpack-otel

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/pipeline.xpack.otel.yml`](../x-pack/pipeline.xpack.otel.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/pipeline.xpack.otel.yml`](../x-pack/pipeline.xpack.otel.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-xpack-otel**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| check / x-pack/otel: Run check | agents `{"cpu": "4000m", "ephemeralStorage": "50G", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "12Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `x-pack-otel-check` | Configured step; details in source |
| Mandatory Tests / :ubuntu: x-pack/otel: Build OTel Collector Distribution | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; key `x-pack-otel-mandatory-build-distro`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/otel: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.11.0`; artifacts `["x-pack/otel/build/*.xml", "x-pack/otel/build/*.json"]`; key `x-pack-otel-mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: x-pack/otel: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; artifacts `["x-pack/otel/build/*.xml", "x-pack/otel/build/*.json"]`; key `x-pack-otel-extended-arm64-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `GCP_DEFAULT_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `x-pack/otel/build/*.json`, `x-pack/otel/build/*.xml` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/pipeline.xpack.otel.yml`](../x-pack/pipeline.xpack.otel.yml)
