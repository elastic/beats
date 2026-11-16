# beats-xpack-dockerlogbeat

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml`](../x-pack/pipeline.xpack.dockerlogbeat.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml`](../x-pack/pipeline.xpack.dockerlogbeat.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-xpack-dockerlogbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / x-pack/dockerlogbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / x-pack/dockerlogbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `x-pack-dockerlogbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: x-pack/dockerlogbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/qemu.sh`; artifacts `["x-pack/dockerlogbeat/build/*.xml", "x-pack/dockerlogbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/dockerlogbeat: Go Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/dockerlogbeat/build/*.xml", "x-pack/dockerlogbeat/build/*.json", "x-pack/dockerlogbeat/build/integration-tests/*", "x-pack/dockerlogbeat/build/integration-tests/Test*/*", "x-pack/dockerlogbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/dockerlogbeat: Go Integration Tests | agents `{"image": "${IMAGE_WIN_2022}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["x-pack/dockerlogbeat/build/*.xml", "x-pack/dockerlogbeat/build/*.json", "x-pack/dockerlogbeat/build/integration-tests/*", "x-pack/dockerlogbeat/build/integration-tests/Test*/*", "x-pack/dockerlogbeat/build/integration-tests/Test*/data/**/*"]`; key `mandatory-int-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 4 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `["xpack-dockerlogbeat-mandatory-tests"]` | Configured step; details in source |
| Packaging / :ubuntu: x-pack/dockerlogbeat: Packaging Linux | condition `build.pull_request.id != null`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `auditbeat-package-linux-x86`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/dockerlogbeat: Packaging arm64 | condition `build.pull_request.id != null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `auditbeat-package-linux-arm`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_2022`, `PLATFORMS`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `x-pack/dockerlogbeat/build/*.json`, `x-pack/dockerlogbeat/build/*.xml`, `x-pack/dockerlogbeat/build/integration-tests/*`, `x-pack/dockerlogbeat/build/integration-tests/Test*/*`, `x-pack/dockerlogbeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml`](../x-pack/pipeline.xpack.dockerlogbeat.yml)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
