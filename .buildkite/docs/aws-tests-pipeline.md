# Aws Tests Pipeline

Documents the Buildkite pipeline configuration in [`.buildkite/aws-tests-pipeline.yml`](../aws-tests-pipeline.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/aws-tests-pipeline.yml`](../aws-tests-pipeline.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured test steps for **Aws Tests Pipeline**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| :ubuntu: x-pack/filebeat: AWS Tests | agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/initCloudEnv.sh`; plugins `elastic/oblt-aws-auth#v0.2.0`; artifacts `["x-pack/filebeat/build/*.xml", "x-pack/filebeat/build/*.json"]`; key `x-pack-filebeat-extended-cloud-test` |
| :ubuntu: x-pack/metricbeat: AWS Tests | agents `{"image": "${AWS_IMAGE_UBUNTU_ARM_64}", "instanceType": "${AWS_ARM_INSTANCE_TYPE}", "provider": "aws"}` | None declared | runs `.buildkite/scripts/initCloudEnv.sh`; plugins `elastic/oblt-aws-auth#v0.2.0`; artifacts `["x-pack/metricbeat/build/*.xml", "x-pack/metricbeat/build/*.json"]`; key `x-pack-metricbeat-extended-cloud-test` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `ASDF_MAGE_VERSION`, `ASDF_TERRAFORM_VERSION`, `AWS_ARM_INSTANCE_TYPE`, `AWS_IMAGE_UBUNTU_ARM_64`, `MODULE`, `MODULE_DIR`, `RACE_DETECTOR`, `TEST_COVERAGE`, `TEST_TAGS` |
| Outputs | `x-pack/filebeat/build/*.json`, `x-pack/filebeat/build/*.xml`, `x-pack/metricbeat/build/*.json`, `x-pack/metricbeat/build/*.xml` |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/aws-tests-pipeline.yml`](../aws-tests-pipeline.yml)
- Referenced local file: [`.buildkite/deploy/docker/docker-compose.yml`](../deploy/docker/docker-compose.yml)
- Referenced local file: [`.buildkite/scripts/initCloudEnv.sh`](../scripts/initCloudEnv.sh)
