# beats-pkg-systemmetrics

Documents the Buildkite pipeline configuration in [`.buildkite/pkg/systemmetrics/pipeline.yml`](../pkg/systemmetrics/pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/pkg/systemmetrics/pipeline.yml`](../pkg/systemmetrics/pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **beats-pkg-systemmetrics**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Check/Update / pkg/systemmetrics: Run check | agents `{"cpu": "4000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "8Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command; retry policy `{"automatic": [{"limit": 1}]}` |
| Step 2 | condition `build.env("BUILDKITE_PULL_REQUEST") != "false"` | `systemmetrics-check-update` | Configured step; details in source |
| Mandatory Tests / :ubuntu: pkg/systemmetrics: Ubuntu x86_64 Unit Tests | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["pkg/systemmetrics/build/*.xml", "pkg/systemmetrics/build/*.json"]`; key `mandatory-systemmetrics-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :windows: pkg/systemmetrics: Win 2022 Unit Tests | agents `{"disk_size": 100, "disk_type": "pd-ssd", "image": "${IMAGE_WIN_2022}", "machine_type": "${GCP_WIN_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; artifacts `["pkg/systemmetrics/build/*.xml", "pkg/systemmetrics/build/*.json"]`; key `mandatory-systemmetrics-win-2022-unit-test`; retry policy `{"automatic": [{"limit": 1}]}` |
| Mandatory Tests / :linux: pkg/systemmetrics: Container-System Tests -- {{matrix.image}} | agents `{"image": "{{matrix.image}}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command; plugins `test-collector#v1.10.2`; artifacts `["pkg/systemmetrics/build/*.xml", "pkg/systemmetrics/build/*.json"]`; key `mandatory-systemmetrics-container-tests`; matrix `{"setup": {"image": ["${IMAGE_UBUNTU_X86_64}", "${IMAGE_RHEL9_X86_64}", "${IMAGE_UBUNTU_2004_X86_64}"]}}`; retry policy `{"automatic": [{"limit": 1}]}` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `GCP_DEFAULT_MACHINE_TYPE`, `GCP_WIN_MACHINE_TYPE`, `HOST_OS_IMAGE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_RHEL9_X86_64`, `IMAGE_UBUNTU_2004_X86_64`, `IMAGE_UBUNTU_X86_64`, `IMAGE_WIN_2022`, `RACE_DETECTOR`, `TEST_COVERAGE` |
| Outputs | `pkg/systemmetrics/build/*.json`, `pkg/systemmetrics/build/*.xml` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/pkg/systemmetrics/pipeline.yml`](../pkg/systemmetrics/pipeline.yml)
