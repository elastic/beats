# beats-filebeat

Documents the Buildkite pipeline configuration in [`.buildkite/filebeat/filebeat-pipeline.yml`](../filebeat/filebeat-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/filebeat/filebeat-pipeline.yml`](../filebeat/filebeat-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-filebeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / Filebeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / Filebeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `filebeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: Filebeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Ubuntu x86_64 Unit Tests with requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Go Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Filebeat: Go Integration Tests | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Go fips140=only Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Python Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/integration-tests/*", "filebeat/build/integration-tests/Test*/*", "filebeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Filebeat: Log input as Filestream Python Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/integration-tests/*", "filebeat/build/integration-tests/Test*/*", "filebeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Filebeat: Win 2016 Unit Tests | agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/integration-tests/*", "filebeat/build/integration-tests/Test*/*", "filebeat/build/integration-tests/Test*/data/**/*"]`; key `windows-2016-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Filebeat: Win 2022 Unit Tests | agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json"]`; key `windows-2022-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: Filebeat: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json", "filebeat/build/input-test/**", "filebeat/build/integration-tests/**"]`; key `extended-arm64-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Filebeat: Win 2025 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2025}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json"]`; key `windows-extended-2025`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Filebeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json"]`; key `windows-extended-2019`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Filebeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json"]`; key `windows-extended-11`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Filebeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["filebeat/build/*.xml", "filebeat/build/*.json"]`; key `windows-extended-10`; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 6 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `filebeat-mandatory-tests` | Configured step; details in source |
| Packaging / :ubuntu: Filebeat: Packaging Linux | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: Filebeat: Packaging arm64 | agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_KIND_VERSION`, `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `IMAGE_WIN_2025`, `K8S_VERSION`, `PLATFORMS`, `PYTEST_ADDOPTS`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `filebeat/build/*.json`, `filebeat/build/*.xml`, `filebeat/build/input-test/**`, `filebeat/build/integration-tests/*`, `filebeat/build/integration-tests/**`, `filebeat/build/integration-tests/Test*/*`, `filebeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/filebeat/filebeat-pipeline.yml`](../filebeat/filebeat-pipeline.yml)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
