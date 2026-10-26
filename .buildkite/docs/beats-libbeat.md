# beats-libbeat

Documents the Buildkite pipeline configuration in [`.buildkite/libbeat/pipeline.libbeat.yml`](../libbeat/pipeline.libbeat.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/libbeat/pipeline.libbeat.yml`](../libbeat/pipeline.libbeat.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-libbeat**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / Libbeat: Run check/update | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Check/Update / Libbeat: Run pre-commit | agents `{"image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "2Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `libbeat-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: Libbeat: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `mandatory-linux-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Libbeat: Ubuntu x86_64 Go Unit Tests with fips provider and requirefips build tag | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-tag`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Libbeat: Ubuntu x86_64 fips140=only Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `mandatory-linux-unit-test-fips-only`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Libbeat: Go Integration Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json", "libbeat/build/integration-tests/*", "libbeat/build/integration-tests/Test*/*", "libbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Libbeat: Go Integration Tests | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"`; agents `{"disk_size": 200, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json", "libbeat/build/integration-tests/*", "libbeat/build/integration-tests/Test*/*", "libbeat/build/integration-tests/Test*/data/**/*"]`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :ubuntu: Libbeat: Stress Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `libbeat/libbeat-stress-test.xml`; key `mandatory-stress-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Libbeat: Win 2016 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2016}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `mandatory-win-2016-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: Libbeat: Win 2022 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `mandatory-win-2022-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Tests / :ubuntu: Libbeat: Ubuntu arm64 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*arm.*/`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `extended-arm64-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Libbeat: Win 10 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_10}", "imageVersion": "${IMAGE_WIN_10_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `extended-win-10-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Libbeat: Win 11 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"diskSizeGb": "256", "image": "${IMAGE_WIN_11}", "imageVersion": "${IMAGE_WIN_11_VERSION}", "provider": "azure", "vmSize": "Standard_D8s_v5"}` | None declared | runs configured command; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `extended-win-11-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |
| Extended Windows Tests / :windows: Libbeat: Win 2019 Unit Tests | condition `build.env("BUILDKITE_PULL_REQUEST") == "false" \|\| build.env("GITHUB_PR_LABELS") =~ /.*[Ww]indows.*/`; agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2019}", "machineType": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["libbeat/build/*.xml", "libbeat/build/*.json"]`; key `extended-win-2019-unit-tests`; retry policy `{"automatic": [{"limit": 1}]}` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_HI_PERF_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_10`, `IMAGE_WIN_10_VERSION`, `IMAGE_WIN_11`, `IMAGE_WIN_11_VERSION`, `IMAGE_WIN_2016`, `IMAGE_WIN_2019`, `IMAGE_WIN_2022`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `libbeat/build/*.json`, `libbeat/build/*.xml`, `libbeat/build/integration-tests/*`, `libbeat/build/integration-tests/Test*/*`, `libbeat/build/integration-tests/Test*/data/**/*`, `libbeat/libbeat-stress-test.xml` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/libbeat/pipeline.libbeat.yml`](../libbeat/pipeline.libbeat.yml)
