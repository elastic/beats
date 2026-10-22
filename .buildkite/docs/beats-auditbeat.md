# beats-auditbeat

Documents the Buildkite pipeline configuration in [`.buildkite/auditbeat/auditbeat-pipeline.yml`](../auditbeat/auditbeat-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/auditbeat/auditbeat-pipeline.yml`](../auditbeat/auditbeat-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-auditbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / Auditbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / Auditbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `auditbeat-check-update` | Configured step; details in source |
| Mandatory Testing / :ubuntu: Auditbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_2204_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :ubuntu: Auditbeat: Ubuntu x86_64 Unit Tests with requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_2204_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :ubuntu: Auditbeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_2204_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :rhel: Auditbeat: RHEL9 Unit Tests | agents `{"image": "${IMAGE_RHEL9}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :windows: Auditbeat: Win 2016 Unit Tests | agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :windows: Auditbeat: Win 2022 Unit Tests | agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Testing / :windows: Auditbeat: Win 2025 Unit Tests | agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2025}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: Auditbeat: Ubuntu x86_64 Integration Tests -- {{matrix.image}} | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*integrations.*/`; agents `{"image": "{{matrix.image}}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-integ-tests`; matrix `{"setup": {"image": ["${IMAGE_UBUNTU_2004_X86_64}", "${IMAGE_UBUNTU_2204_X86_64}", "${IMAGE_UBUNTU_2404_X86_64}"]}}`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: Auditbeat: Ubuntu arm64 Integration Tests -- {{matrix.image}} | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*integrations.*/`; agents `{"image": "{{matrix.image}}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-arm64-integ-tests`; matrix `{"setup": {"image": ["${IMAGE_UBUNTU_2004_ARM64}", "${IMAGE_UBUNTU_2204_ARM64}", "${IMAGE_UBUNTU_2404_ARM64}"]}}`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: Auditbeat: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${IMAGE_UBUNTU_2204_ARM64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-arm64-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Auditbeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-win-2019-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Auditbeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-win-10-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Auditbeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["auditbeat/build/*.xml", "auditbeat/build/*.json"]`; key `auditbeat-extended-win-11-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 6 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `auditbeat-mandatory-tests` | Configured step; details in source |
| Packaging / :ubuntu: Auditbeat: Packaging Linux | condition `build.pull_request.id != null`; agents `{"image": "${IMAGE_UBUNTU_2204_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `auditbeat-package-linux-x86`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |
| Packaging / :ubuntu: Auditbeat: Packaging Linux arm64 | condition `build.pull_request.id != null`; agents `{"image": "${IMAGE_UBUNTU_2204_ARM64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/packaging/packaging.sh`; plugins `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; key `auditbeat-package-linux-arm64`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 20 minutes |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_RHEL9`, `IMAGE_UBUNTU_2004_ARM64`, `IMAGE_UBUNTU_2004_X86_64`, `IMAGE_UBUNTU_2204_ARM64`, `IMAGE_UBUNTU_2204_X86_64`, `IMAGE_UBUNTU_2404_ARM64`, `IMAGE_UBUNTU_2404_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `IMAGE_WIN_2025`, `PLATFORMS`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `auditbeat/build/*.json`, `auditbeat/build/*.xml` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/auditbeat/auditbeat-pipeline.yml`](../auditbeat/auditbeat-pipeline.yml)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
