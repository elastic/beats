---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/winlogbeat-modules.html
---

# Modules [winlogbeat-modules]

::::{note}
Winlogbeat modules have changed in 8.0.0 to use Elasticsearch Ingest Node for processing. If you are upgrading from 7.x please review the documentation and see the default configuration file.
::::


This section contains detailed information about the available Windows event log processing modules contained in Winlogbeat. More details about each module can be found in the module’s documentation.

Winlogbeat modules are implemented using Elasticsearch Ingest Node pipelines. The events receive their transformations within Elasticsearch. All events are sent through Winlogbeat’s "routing" pipeline that routes events to specific module pipelines based on their `winlog.channel` value.

Winlogbeat’s default config file contains the option to send all events to the routing pipeline. If you remove this option then the module processing will not be applied.

```yaml
output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing
```

The general goal of each module is to transform events by renaming fields to comply with the [Elastic Common Schema](ecs://reference/index.md) (ECS). The modules may also apply additional categorization, tagging, and parsing as necessary.

::::{note}
The provided modules only support events in English. For more information about how to configure the language in `winlogbeat`, refer to [Winlogbeat](/reference/winlogbeat/configuration-winlogbeat-options.md).
::::



## Setup of Ingest Node pipelines [winlogbeat-modules-setup]

Winlogbeat’s Ingest Node pipelines must be installed to Elasticsearch if you want to apply the module processing to events. The simplest way to get started is to use the Elasticsearch output and Winlogbeat will automatically install the pipelines when it first connects to Elasticsearch.

Installation Methods

1. [On connection to {{es}}](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-auto)
2. [setup command](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-setup)
3. [Manually install pipelines](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-manual)


## Usage with Forwarded Events [_usage_with_forwarded_events]

No special configuration options are required when working with the `ForwardedEvents` channel. The events in this log retain the channel name of their origin (e.g. `winlog.channel: Security`). And because the routing pipeline processes events based on the channel name no special config is necessary.

```yaml
winlogbeat.event_logs:
- name: ForwardedEvents
  tags: [forwarded]
  language: 0x0409 # en-US

output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing
```


## Modules [_modules]

* [Powershell](/reference/winlogbeat/winlogbeat-module-powershell.md)
* [Security](/reference/winlogbeat/winlogbeat-module-security.md)
* [Sysmon](/reference/winlogbeat/winlogbeat-module-sysmon.md)

