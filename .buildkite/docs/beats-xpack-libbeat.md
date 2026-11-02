# beats-xpack-libbeat

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/pipeline.xpack.libbeat.yml`](../x-pack/pipeline.xpack.libbeat.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/pipeline.xpack.libbeat.yml`](../x-pack/pipeline.xpack.libbeat.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-xpack-libbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / x-pack/libbeat: Run check/update | agents `{"cpu": "4000m", "ephemeralStorage": "40G", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / x-pack/libbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `x-pack-libbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: x-pack/libbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/libbeat: Ubuntu x86_64 Unit Tests with requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-tag`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/libbeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-only`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/libbeat: Go Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json", "x-pack/libbeat/build/integration-tests/*", "x-pack/libbeat/build/integration-tests/Test*/*", "x-pack/libbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/libbeat: Go Integration Tests | agents `{"image": "${IMAGE_WIN_2022}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json", "x-pack/libbeat/build/integration-tests/*", "x-pack/libbeat/build/integration-tests/Test*/*", "x-pack/libbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/libbeat: Python Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json", "x-pack/libbeat/build/integration-tests/*", "x-pack/libbeat/build/integration-tests/Test*/*", "x-pack/libbeat/build/integration-tests/Test*/data/**/*"]`; key `mandatory-python-int-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: x-pack/libbeat: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; artifacts `["x-pack/libbeat/build/*.xml", "x-pack/libbeat/build/*.json"]`; key `extended-arm64-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_2022`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `x-pack/libbeat/build/*.json`, `x-pack/libbeat/build/*.xml`, `x-pack/libbeat/build/integration-tests/*`, `x-pack/libbeat/build/integration-tests/Test*/*`, `x-pack/libbeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/pipeline.xpack.libbeat.yml`](../x-pack/pipeline.xpack.libbeat.yml)
