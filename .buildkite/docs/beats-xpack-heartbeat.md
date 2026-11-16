# beats-xpack-heartbeat

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/pipeline.xpack.heartbeat.yml`](../x-pack/pipeline.xpack.heartbeat.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/pipeline.xpack.heartbeat.yml`](../x-pack/pipeline.xpack.heartbeat.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-xpack-heartbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / x-pack/heartbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / x-pack/heartbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `x-pack-heartbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: x-pack/heartbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/heartbeat: Go Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json", "x-pack/heartbeat/build/integration-tests/*", "x-pack/heartbeat/build/integration-tests/Test*/*", "x-pack/heartbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/heartbeat: Go Integration Tests | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json", "x-pack/heartbeat/build/integration-tests/*", "x-pack/heartbeat/build/integration-tests/Test*/*", "x-pack/heartbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/heartbeat: Win 2016 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `mandatory-win-2016-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/heartbeat: Win 2022 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `mandatory-win-2022-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: x-pack/heartbeat: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `extended-arm64-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/heartbeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `extended-win-10-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/heartbeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `extended-win-11-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/heartbeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machineType": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/heartbeat/build/*.xml", "x-pack/heartbeat/build/*.json"]`; key `extended-win-2019-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 6 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `[{"step": "x-pack-heartbeat-mandatory-tests"}]` | Configured step; details in source |
| Packaging / :ubuntu: x-pack/heartbeat: Packaging Linux | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/heartbeat: Packaging arm64 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/heartbeat: Packaging Linux amd64 FIPS | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/heartbeat: Packaging Linux arm64 FIPS | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `ASDF_NODEJS_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `ELASTIC_SYNTHETICS_CAPABLE`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `HEARTBEAT_REQUIRE_BROWSER_E2E_ARCHIVE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_RHEL9_X86_64`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `PLATFORMS`, `PLATFORMS_AMD64_FIPS`, `PLATFORMS_ARM64_FIPS`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `x-pack/heartbeat/build/*.json`, `x-pack/heartbeat/build/*.xml`, `x-pack/heartbeat/build/integration-tests/*`, `x-pack/heartbeat/build/integration-tests/Test*/*`, `x-pack/heartbeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/pipeline.xpack.heartbeat.yml`](../x-pack/pipeline.xpack.heartbeat.yml)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
