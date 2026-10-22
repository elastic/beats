# Deploy K8S Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/deploy/kubernetes/deploy-k8s-pipeline.yml`](../deploy/kubernetes/deploy-k8s-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/deploy/kubernetes/deploy-k8s-pipeline.yml`](../deploy/kubernetes/deploy-k8s-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured deployment steps for **Deploy K8S Pipeline**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Deploy/K8S / Deploy/K8S: Run pre-commit | agents `{"cpu": "2000m", "image": "${IMAGE_BEATS_WITH_HOOKS_LATEST}", "memory": "4Gi", "useCustomGlobalHooks": true}` | None declared | runs configured command |
| Deploy/K8S / Checks | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | runs configured command |
| Deploy/K8S / K8S Test/K8S version: {{matrix}} | agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_HI_PERF_MACHINE_TYPE}", "provider": "gcp"}` | None declared | matrix `["v1.36.1", "v1.35.0", "v1.34.0", "v1.33.0"]`; retry policy `{"automatic": [{"limit": 1}]}` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_KIND_VERSION`, `ASDF_MAGE_VERSION`, `GCP_HI_PERF_MACHINE_TYPE`, `IMAGE_BEATS_WITH_HOOKS_LATEST`, `IMAGE_UBUNTU_X86_64`, `K8S_VERSION`, `MODULE` |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/deploy/kubernetes/deploy-k8s-pipeline.yml`](../deploy/kubernetes/deploy-k8s-pipeline.yml)
- Referenced local file: [`.buildkite/deploy/kubernetes/scripts/kind-setup.sh`](../deploy/kubernetes/scripts/kind-setup.sh)
- Referenced local file: [`.buildkite/deploy/kubernetes/scripts/setup-k8s-env.sh`](../deploy/kubernetes/scripts/setup-k8s-env.sh)
