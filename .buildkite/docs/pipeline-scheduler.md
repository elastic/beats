# Pipeline Scheduler

Documents the Buildkite pipeline configuration in [`.buildkite/pipeline-scheduler.yml`](../pipeline-scheduler.yml).

## At a glance

| | |
| --- | --- |
| Platform | Buildkite |
| Source | [`.buildkite/pipeline-scheduler.yml`](../pipeline-scheduler.yml) |
| Trigger | Schedule-driven configuration; current timing is not specified here. |

## Purpose

Uses the selected pipeline names and active release branches to generate Buildkite trigger steps dynamically. The checked-in source does not state a separate historical rationale.

## When it runs

This pipeline is used for scheduled builds. The checked-in YAML does not record the live scheduler cadence; consult the Buildkite pipeline schedule settings. The [`pipeline-scheduler.yml`](../pipeline-scheduler.yml) and [`pipeline-scheduler.py`](../pipeline-scheduler.py) sources generate scheduled branch-trigger steps where configured.

## Execution flow

| Step | Runs when / agent | Depends on | Configured work |
| --- | --- | --- | --- |
| :pipeline: Generate trigger steps for $PIPELINES_TO_TRIGGER | agents `{"image": "docker.elastic.co/ci-agent-images/platform-ingest/buildkite-agent-beats-ci-with-hooks:0.1", "useCustomGlobalHooks": true}` | None declared | runs `.buildkite/pipeline-scheduler.py` |

## Inputs, outputs, and access

| Item | Details |
| --- | --- |
| Inputs | `PIPELINES_TO_TRIGGER` |
| Outputs | No artifact paths declared directly in this YAML. |
| Permissions | Not specified in the checked-in YAML; agent and plugin configuration is listed with the steps above. |
| Secrets | No secret-like variable references identified in this YAML; runtime secrets may be injected outside the file. |

## Operational notes

Only the checked-in pipeline YAML and referenced local scripts are described here. Runtime-generated steps and remote pipeline settings are not visible unless explicitly produced by the source.

## Sources

- Pipeline: [`.buildkite/pipeline-scheduler.yml`](../pipeline-scheduler.yml)
- Referenced local file: [`.buildkite/pipeline-scheduler.py`](../pipeline-scheduler.py)
