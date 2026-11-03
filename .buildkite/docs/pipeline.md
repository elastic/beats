# Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/pipeline.yml`](../pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/pipeline.yml`](../pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Routes pull-request changes to beat-specific Buildkite pipelines and triggers the serverless Beats test pipeline for non-pull-request builds. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Trigger Auditbeat | condition `build.pull_request.id != null`; changed paths include `["auditbeat/**", ".buildkite/auditbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `auditbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Heartbeat | condition `build.pull_request.id != null`; changed paths include `["heartbeat/**", ".buildkite/heartbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `heartbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Filebeat | condition `build.pull_request.id != null`; changed paths include `["filebeat/**", ".buildkite/filebeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `filebeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/filebeat | condition `build.pull_request.id != null`; changed paths include `["filebeat/**", "x-pack/filebeat/**", ".buildkite/x-pack/pipeline.xpack.filebeat.yml", "x-pack/libbeat/**", "x-pack/otel/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**", ".buildkite/deploy/docker/docker-compose.yml"]` | None declared | triggers pipeline `beats-xpack-filebeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/dockerlogbeat | condition `build.pull_request.id != null`; changed paths include `["x-pack/dockerlogbeat/**", ".buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-dockerlogbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Metricbeat | condition `build.pull_request.id != null`; changed paths include `["metricbeat/**", ".buildkite/metricbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-metricbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/metricbeat | condition `build.pull_request.id != null`; changed paths include `["metricbeat/**", "x-pack/metricbeat/**", ".buildkite/x-pack/pipeline.xpack.metricbeat.yml", "x-pack/libbeat/**", "x-pack/otel/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**", ".buildkite/deploy/docker/docker-compose.yml"]` | None declared | triggers pipeline `beats-xpack-metricbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/osquerybeat | condition `build.pull_request.id != null`; changed paths include `["x-pack/osquerybeat/**", ".buildkite/x-pack/pipeline.xpack.osquerybeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-osquerybeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/winlogbeat | condition `build.pull_request.id != null`; changed paths include `["winlogbeat/**", "x-pack/winlogbeat/**", ".buildkite/x-pack/pipeline.xpack.winlogbeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-winlogbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Deploy/K8S | condition `build.pull_request.id != null`; changed paths include `["deploy/kubernetes/**", "metricbeat/module/kubernetes/**", ".go-version", "libbeat/docs/version.asciidoc", ".buildkite/deploy/kubernetes/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `deploy-k8s`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Libbeat | condition `build.pull_request.id != null`; changed paths include `["libbeat/**", ".buildkite/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-libbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger pkg/systemmetrics | condition `build.pull_request.id != null`; changed paths include `["pkg/systemmetrics/**", ".buildkite/pkg/systemmetrics/**", ".go-version", "go.mod", "dev-tools/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-pkg-systemmetrics`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/libbeat | condition `build.pull_request.id != null`; changed paths include `["x-pack/libbeat/**", ".buildkite/x-pack/pipeline.xpack.libbeat.yml", ".go-version", "libbeat/**", "go.mod", "pytest.ini", "dev-tools/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-libbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/otel | condition `build.pull_request.id != null`; changed paths include `["x-pack/otel/**", ".buildkite/x-pack/pipeline.xpack.otel.yml", ".go-version", "go.mod", "dev-tools/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-otel`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/auditbeat | condition `build.pull_request.id != null`; changed paths include `["auditbeat/**", "x-pack/auditbeat/**", ".buildkite/x-pack/pipeline.xpack.auditbeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-auditbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/heartbeat | condition `build.pull_request.id != null`; changed paths include `["heartbeat/**", "x-pack/heartbeat/**", ".buildkite/x-pack/pipeline.xpack.heartbeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-heartbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger x-pack/packetbeat | condition `build.pull_request.id != null`; changed paths include `["packetbeat/**", "x-pack/packetbeat/**", ".buildkite/x-pack/pipeline.xpack.packetbeat.yml", "x-pack/libbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-xpack-packetbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Winlogbeat | condition `build.pull_request.id != null`; changed paths include `["winlogbeat/**", ".buildkite/winlogbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-winlogbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Trigger Packetbeat | condition `build.pull_request.id != null`; changed paths include `["packetbeat/**", ".buildkite/packetbeat/**", ".go-version", "go.mod", "pytest.ini", "dev-tools/**", "libbeat/**", "pkg/systemmetrics/**", "testing/**", ".buildkite/pipeline.yml", ".buildkite/scripts/**", ".buildkite/hooks/**"]` | None declared | triggers pipeline `beats-packetbeat`; trigger build fields `["commit", "branch", "env", "env.BUILDKITE_PULL_REQUEST", "env.BUILDKITE_PULL_REQUEST_BASE_BRANCH", "env.GITHUB_PR_LABELS"]` |
| Serverless beats tests(elastic-agent) | condition `build.pull_request.id == null` | None declared | triggers pipeline `beats-agent-serverless-tests`; trigger build fields `["branch"]` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `BUILDKITE_BRANCH`, `BUILDKITE_COMMIT`, `BUILDKITE_PULL_REQUEST`, `BUILDKITE_PULL_REQUEST_BASE_BRANCH`, `GITHUB_PR_LABELS` |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/pipeline.yml`](../pipeline.yml)
- Referenced local file: [`.buildkite/deploy/docker/docker-compose.yml`](../deploy/docker/docker-compose.yml)
- Referenced local file: [`.buildkite/scripts/custom_fips_ech_test.sh`](../scripts/custom_fips_ech_test.sh)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/ech.sh`](../scripts/ech.sh)
- Referenced local file: [`.buildkite/scripts/initCloudEnv.sh`](../scripts/initCloudEnv.sh)
- Referenced local file: [`.buildkite/scripts/install_sysmon.ps1`](../scripts/install_sysmon.ps1)
- Referenced local file: [`.buildkite/scripts/packaging/packaging.sh`](../scripts/packaging/packaging.sh)
- Referenced local file: [`.buildkite/scripts/qemu.sh`](../scripts/qemu.sh)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.auditbeat.yml`](../x-pack/pipeline.xpack.auditbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.dockerlogbeat.yml`](../x-pack/pipeline.xpack.dockerlogbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.filebeat.yml`](../x-pack/pipeline.xpack.filebeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.heartbeat.yml`](../x-pack/pipeline.xpack.heartbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.libbeat.yml`](../x-pack/pipeline.xpack.libbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.metricbeat.yml`](../x-pack/pipeline.xpack.metricbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.osquerybeat.yml`](../x-pack/pipeline.xpack.osquerybeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.otel.yml`](../x-pack/pipeline.xpack.otel.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.packetbeat.yml`](../x-pack/pipeline.xpack.packetbeat.yml)
- Referenced local file: [`.buildkite/x-pack/pipeline.xpack.winlogbeat.yml`](../x-pack/pipeline.xpack.winlogbeat.yml)
