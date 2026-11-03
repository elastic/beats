# Beats Tests Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured test steps for **Beats Tests Pipeline**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Triggering Build for Auditbeat | No step-level condition or agent declared | None declared | triggers pipeline `auditbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Heartbeat | No step-level condition or agent declared | None declared | triggers pipeline `heartbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Filebeat | No step-level condition or agent declared | None declared | triggers pipeline `filebeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/filebeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-filebeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/dockerlogbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-dockerlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Metricbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-metricbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/metricbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-metricbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/osquerybeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-osquerybeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/winlogbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-winlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Deploy/K8S | No step-level condition or agent declared | None declared | triggers pipeline `deploy-k8s`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Libbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-libbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for pkg/systemmetrics | No step-level condition or agent declared | None declared | triggers pipeline `beats-pkg-systemmetrics`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/libbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-libbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/otel | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-otel`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/auditbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-auditbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/heartbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-heartbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for x-pack/packetbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-xpack-packetbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Winlogbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-winlogbeat`; trigger build fields `["commit", "branch"]` |
| Triggering Build for Packetbeat | No step-level condition or agent declared | None declared | triggers pipeline `beats-packetbeat`; trigger build fields `["commit", "branch"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `BUILDKITE_BRANCH`, `BUILDKITE_COMMIT` |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/beats-tests-pipeline.yml`](../beats-tests-pipeline.yml)
