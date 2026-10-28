# beats-metricbeat

Documents the Buildkite pipeline configuration in [`.buildkite/metricbeat/pipeline.yml`](../metricbeat/pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/metricbeat/pipeline.yml`](../metricbeat/pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-metricbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / Metricbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / Metricbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `metricbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: Metricbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Metricbeat: Ubuntu x86_64 Unit Tests with requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-tag`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Metricbeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-only`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Metricbeat: Go Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/deploy/kubernetes/scripts/setup-k8s-env.sh`; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json", "metricbeat/build/integration-tests/*", "metricbeat/build/integration-tests/Test*/*", "metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Metricbeat: Go Integration Tests (Module) | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json", "metricbeat/build/integration-tests/*", "metricbeat/build/integration-tests/Test*/*", "metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Metricbeat: Go fips140=only Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/deploy/kubernetes/scripts/setup-k8s-env.sh`; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json", "metricbeat/build/integration-tests/*", "metricbeat/build/integration-tests/Test*/*", "metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Metricbeat: Python Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/deploy/kubernetes/scripts/setup-k8s-env.sh`; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json", "metricbeat/build/integration-tests/*", "metricbeat/build/integration-tests/Test*/*", "metricbeat/build/integration-tests/Test*/data/**/*"]`; key `mandatory-python-int-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Metricbeat: Win 2016 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `mandatory-win-2016-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Metricbeat: Win 2022 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `mandatory-win-2022-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Metricbeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `extended-win-10-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Metricbeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `extended-win-11-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Metricbeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machineType": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `extended-win-2019-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Metricbeat: Win 2025 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2025}", "machineType": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["metricbeat/build/*.xml", "metricbeat/build/*.json"]`; key `extended-win-2025-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 5 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `metricbeat-mandatory-tests` | Configured step; details in source |
| Packaging / :ubuntu: Metricbeat: Packaging Linux | condition `build.pull_request.id != null`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: Metricbeat: Packaging Linux arm64 | condition `build.pull_request.id != null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_KIND_VERSION`, `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `BEAT_PATH`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `IMAGE_WIN_2025`, `K8S_VERSION`, `PLATFORMS`, `PWD`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `metricbeat/build/*.json`, `metricbeat/build/*.xml`, `metricbeat/build/integration-tests/*`, `metricbeat/build/integration-tests/Test*/*`, `metricbeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/metricbeat/pipeline.yml`](../metricbeat/pipeline.yml)
- Referenced local file: [`.buildkite/deploy/kubernetes/scripts/kind-setup.sh`](../deploy/kubernetes/scripts/kind-setup.sh)
- Referenced local file: [`.buildkite/deploy/kubernetes/scripts/setup-k8s-env.sh`](../deploy/kubernetes/scripts/setup-k8s-env.sh)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
