# Packaging.Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/packaging.pipeline.yml`](../packaging.pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/packaging.pipeline.yml`](../packaging.pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Builds and stages Beats packages for snapshot or staging publication, including dashboards and DRA preparation. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| start-gate-snapshot | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null` | None declared | runs configured command; key `start-gate-snapshot` |
| start-gate-staging | condition `build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null` | None declared | runs configured command; key `start-gate-staging` |
| Beats dashboards / Snapshot dashboards | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-snapshot` | plugins `elastic/oblt-google-auth#v1.3.1`; artifacts `["build/distributions/**/*"]`; key `dashboards-snapshot`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Beats dashboards / Staging dashboards | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-staging` | plugins `elastic/oblt-google-auth#v1.3.1`; artifacts `["build/distributions/**/*"]`; key `dashboards-staging`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging snapshot / SNAPSHOT: {{matrix}} | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-snapshot` | runs `.buildkite/scripts/packaging/package-dra.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["auditbeat", "filebeat", "heartbeat", "metricbeat", "packetbeat", "winlogbeat", "x-pack/auditbeat", "x-pack/dockerlogbeat", "x-pack/filebeat", "x-pack/heartbeat", "x-pack/metricbeat", "x-pack/osquerybeat", "x-pack/packetbeat", "x-pack/winlogbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging snapshot / SNAPSHOT: {{matrix}} Linux/arm64, Darwin/arm64, and Windows/arm64 | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | `start-gate-snapshot` | runs `.buildkite/scripts/packaging/package-dra.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["auditbeat", "filebeat", "heartbeat", "metricbeat", "packetbeat", "winlogbeat", "x-pack/auditbeat", "x-pack/dockerlogbeat", "x-pack/filebeat", "x-pack/heartbeat", "x-pack/metricbeat", "x-pack/osquerybeat", "x-pack/packetbeat", "x-pack/winlogbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging snapshot / SNAPSHOT: {{matrix}} Linux/amd64 and Windows/amd64 FIPS | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-snapshot` | runs `.buildkite/scripts/packaging/package-dra.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["x-pack/auditbeat", "x-pack/filebeat", "x-pack/metricbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging snapshot / SNAPSHOT: {{matrix}} Linux/arm64 and Windows/arm64 FIPS | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | `start-gate-snapshot` | runs `.buildkite/scripts/packaging/package-dra.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["x-pack/auditbeat", "x-pack/filebeat", "x-pack/metricbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging Staging / STAGING: {{matrix}} | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-staging` | runs `.buildkite/scripts/packaging/package-dra.sh`; runs `.buildkite/scripts/version_qualifier.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["auditbeat", "filebeat", "heartbeat", "metricbeat", "packetbeat", "winlogbeat", "x-pack/auditbeat", "x-pack/dockerlogbeat", "x-pack/filebeat", "x-pack/heartbeat", "x-pack/metricbeat", "x-pack/osquerybeat", "x-pack/packetbeat", "x-pack/winlogbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging Staging / STAGING: {{matrix}} Linux/arm64, Darwin/arm64, and Windows/arm64 | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | `start-gate-staging` | runs `.buildkite/scripts/packaging/package-dra.sh`; runs `.buildkite/scripts/version_qualifier.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["auditbeat", "filebeat", "heartbeat", "metricbeat", "packetbeat", "winlogbeat", "x-pack/auditbeat", "x-pack/dockerlogbeat", "x-pack/filebeat", "x-pack/heartbeat", "x-pack/metricbeat", "x-pack/osquerybeat", "x-pack/packetbeat", "x-pack/winlogbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging Staging / STAGING: {{matrix}} Linux/amd64 and Windows/amd64 FIPS | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"image": "${IMAGE_UBUNTU_X86_64}", "machineType": "${GCP_DEFAULT_MACHINE_TYPE}", "provider": "gcp"}` | `start-gate-staging` | runs `.buildkite/scripts/packaging/package-dra.sh`; runs `.buildkite/scripts/version_qualifier.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["x-pack/auditbeat", "x-pack/filebeat", "x-pack/metricbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| Packaging Staging / STAGING: {{matrix}} Linux/arm64 and Windows/arm64 FIPS | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | `start-gate-staging` | runs `.buildkite/scripts/packaging/package-dra.sh`; runs `.buildkite/scripts/version_qualifier.sh`; plugins `elastic/oblt-google-auth#v1.3.1`, `elastic/vault-docker-login#v0.8.0`, `elastic/vault-docker-login#v0.8.0`; artifacts `["build/distributions/**/*"]`; matrix `["x-pack/auditbeat", "x-pack/filebeat", "x-pack/metricbeat"]`; retry policy `{"automatic": [{"limit": 1}]}`; timeout 40 minutes |
| DRA publish / :pipeline: DRA Prep (snapshot) | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null`; agents `{"provider": "k8s"}` | `["start-gate-snapshot", "packaging-snapshot", "dashboards-snapshot"]` | runs `.buildkite/dra-prep-pipeline.yml`; runs `.buildkite/scripts/packaging/resolve-dra-version.sh`; key `dra-snapshot` |
| DRA publish / :pipeline: DRA Prep (staging) | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null) \|\| build.env('RUN_STAGING') == "true"`; agents `{"provider": "k8s"}` | `["start-gate-staging", "packaging-staging", "dashboards-staging"]` | runs `.buildkite/dra-prep-pipeline.yml`; runs `.buildkite/scripts/packaging/resolve-dra-version.sh`; key `dra-staging` |
| end-gate-snapshot | condition `(build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.branch == 'main' \|\| build.env('RUN_SNAPSHOT') == "true") && build.env('VERSION_QUALIFIER') == null` | None declared | runs configured command; key `end-gate-snapshot` |
| end-gate-staging | condition `build.branch =~ /^[0-9]+\.[0-9x]+\$/ \|\| build.env('VERSION_QUALIFIER') != null` | None declared | runs configured command; key `end-gate-staging` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `BUILDKITE_BRANCH`, `DEV`, `FIPS`, `GCP_DEFAULT_MACHINE_TYPE`, `IMAGE_UBUNTU_X86_64`, `PLATFORMS`, `PLATFORMS_AMD64_FIPS`, `PLATFORMS_ARM`, `PLATFORMS_ARM64_FIPS`, `SNAPSHOT`, `WORKFLOW` |
| Outputs | `build/distributions/**/*` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | Vault-backed plugins are configured; secret values and paths are omitted. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/packaging.pipeline.yml`](../packaging.pipeline.yml)
- Referenced local file: [`.buildkite/dra-prep-pipeline.yml`](../dra-prep-pipeline.yml)
- Referenced local file: [`.buildkite/scripts/docker.sh`](../scripts/docker.sh)
- Referenced local file: [`.buildkite/scripts/dra-annotate.sh`](../scripts/dra-annotate.sh)
- Referenced local file: [`.buildkite/scripts/packaging/package-dra.sh`](../scripts/packaging/package-dra.sh)
- Referenced local file: [`.buildkite/scripts/packaging/resolve-dra-version.sh`](../scripts/packaging/resolve-dra-version.sh)
- Referenced local file: [`.buildkite/scripts/stage-dra-artifacts.sh`](../scripts/stage-dra-artifacts.sh)
- Referenced local file: [`.buildkite/scripts/version_qualifier.sh`](../scripts/version_qualifier.sh)
