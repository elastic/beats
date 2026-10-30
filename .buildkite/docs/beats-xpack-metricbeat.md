# beats-xpack-metricbeat

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/pipeline.xpack.metricbeat.yml`](../x-pack/pipeline.xpack.metricbeat.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/pipeline.xpack.metricbeat.yml`](../x-pack/pipeline.xpack.metricbeat.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-xpack-metricbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / x-pack/metricbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / x-pack/metricbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `x-pack-metricbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Ubuntu x86_64 Unit Tests with requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-tag`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-only`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Go Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json", "x-pack/metricbeat/build/integration-tests/*", "x-pack/metricbeat/build/integration-tests/Test*/*", "x-pack/metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/metricbeat: Go Integration Tests (Module) | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json", "x-pack/metricbeat/build/integration-tests/*", "x-pack/metricbeat/build/integration-tests/Test*/*", "x-pack/metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Go fips140=only Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json", "x-pack/metricbeat/build/integration-tests/*", "x-pack/metricbeat/build/integration-tests/Test*/*", "x-pack/metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: FIPS ECH Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/custom_fips_ech_test.sh`; plugins `test-collector#v1.10.2`, `elastic/vault-secrets#v0.1.0`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json", "x-pack/metricbeat/build/integration-tests/*", "x-pack/metricbeat/build/integration-tests/Test*/*", "x-pack/metricbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: x-pack/metricbeat: Python Integration Tests (Module) | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json", "x-pack/metricbeat/build/integration-tests/*", "x-pack/metricbeat/build/integration-tests/Test*/*", "x-pack/metricbeat/build/integration-tests/Test*/data/**/*"]`; key `mandatory-python-int-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/metricbeat: Win 2016 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `mandatory-win-2016-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: x-pack/metricbeat: Win 2022 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `mandatory-win-2022-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/metricbeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `extended-win-10-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/metricbeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `extended-win-11-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: x-pack/metricbeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machineType": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `extended-win-2019-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: x-pack/metricbeat: AWS Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false"  \|\| build.env("GITHUB_PR_LABELS") =~ /.*(aws).*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/initCloudEnv.sh`; plugins `test-collector#v1.10.2`, `elastic/oblt-aws-auth#v0.2.0`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `x-pack-metricbeat-extended-cloud-test` |
| Step 6 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `[{"step": "x-pack-metricbeat-mandatory-tests"}]` | Configured step; details in source |
| Packaging / :ubuntu: x-pack/metricbeat: Packaging Linux | condition `build.pull_request.id != null`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/metricbeat: Packaging arm64 | condition `build.pull_request.id != null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/metricbeat: Packaging Linux amd64 FIPS | condition `build.pull_request.id != null`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-linux-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: x-pack/metricbeat: Packaging linux arm64 | condition `build.pull_request.id != null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-arm-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :windows: x-pack/metricbeat: Packaging Windows amd64 FIPS | condition `build.pull_request.id != null`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-windows-amd64-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :windows: x-pack/metricbeat: Packaging Windows arm64 FIPS | condition `build.pull_request.id != null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `packaging-windows-arm64-fips`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `ASDF_TERRAFORM_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `BEAT_PATH`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `MODULE`, `MODULE_DIR`, `PLATFORMS`, `PLATFORMS_AMD64_FIPS`, `PLATFORMS_ARM64_FIPS`, `RACE_DETECTOR`, `TEST_COVERAGE`, `TEST_TAGS`, `TF_VAR_ech_region` |
| Outputs | `x-pack/metricbeat/build/*.json`, `x-pack/metricbeat/build/*.xml`, `x-pack/metricbeat/build/integration-tests/*`, `x-pack/metricbeat/build/integration-tests/Test*/*`, `x-pack/metricbeat/build/integration-tests/Test*/data/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/pipeline.xpack.metricbeat.yml`](../x-pack/pipeline.xpack.metricbeat.yml)
- Referenced local file: [`.buildkite/deploy/docker/docker-compose.yml`](../deploy/docker/docker-compose.yml)
- Referenced local file: [`.buildkite/scripts/custom_fips_ech_test.sh`](../scripts/custom_fips_ech_test.sh)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/ech.sh`](../scripts/ech.sh)
- Referenced local file: [`.buildkite/scripts/initCloudEnv.sh`](../scripts/initCloudEnv.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
