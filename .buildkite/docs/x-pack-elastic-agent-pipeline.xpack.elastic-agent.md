# Pipeline.Xpack.Elastic Agent

Documents the Buildkite pipeline configuration in [`.buildkite/x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml`](../x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml`](../x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml) |
| Trigger | Pipeline start conditions are not specified in this YAML. |

## Purpose

Runs the configured Buildkite steps for **Pipeline.Xpack.Elastic Agent**. The checked-in source does not state a separate historical rationale.

## When it runs

Buildkite pipeline triggers and branch filters are not specified in this checked-in YAML. Step-level conditions and path filters, when present, are described below; other activation settings are managed outside this file.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| Example test | No step-level condition or agent declared | None declared | runs configured command |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | No pipeline-level environment input names found in this YAML. |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml`](../x-pack/elastic-agent/pipeline.xpack.elastic-agent.yml)
