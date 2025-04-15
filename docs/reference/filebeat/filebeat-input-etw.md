---
navigation_title: "ETW"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-etw.html
---

# ETW input [filebeat-input-etw]


[Event Tracing for Windows](https://learn.microsoft.com/en-us/windows/win32/etw/event-tracing-portal) is a powerful logging and tracing mechanism built into the Windows operating system. It provides a detailed view of application and system behavior, performance issues, and runtime diagnostics. Trace events contain an event header and provider-defined data that describes the current state of an application or operation. You can use the events to debug an application and perform capacity and performance analysis.

The ETW input can interact with ETW in three distinct ways: it can create a new session to capture events from user-mode providers, attach to an already existing session to collect ongoing event data, or read events from a pre-recorded .etl file. This functionality enables the module to adapt to different scenarios, such as real-time event monitoring or analyzing historical data.

This input currently supports manifest-based, MOF (classic) and TraceLogging providers while WPP providers are not supported. [Here](https://learn.microsoft.com/en-us/windows/win32/etw/about-event-tracing#types-of-providers) you can find more information about the available types of providers.

It has been tested in the Windows versions supported by Filebeat, starting from Windows 10 and Windows Server 2016. In addition, administrative privileges are required to control event tracing sessions.

Example configurations:

Read from a provider by name:

```yaml
filebeat.inputs:
- type: etw
  id: etw-dnsserver
  enabled: true
  provider.name: Microsoft-Windows-DNSServer
  session_name: DNSServer-Analytical
  trace_level: verbose
  match_any_keyword: 0x8000000000000000
  match_all_keyword: 0
```

Read from a provider by its GUID:

```yaml
filebeat.inputs:
- type: etw
  id: etw-dnsserver
  enabled: true
  provider.guid: {EB79061A-A566-4698-9119-3ED2807060E7}
  session_name: DNSServer-Analytical
  trace_level: verbose
  match_any_keyword: 0x8000000000000000
  match_all_keyword: 0
```

Read from an existing session:

```yaml
filebeat.inputs:
- type: etw
  enabled: true
  id: etw-dnsserver-session
  session: UAL_Usermode_Provider
```

Read from a .etl file:

```yaml
filebeat.inputs:
- type: etw
  enabled: true
  id: etw-dnsserver-session
  file: "C\Windows\System32\Winevt\Logs\Logfile.etl"
```

::::{note}
Examples shown above are mutually exclusive, the options `provider.name`, `provider.guid`, `session` and `file` cannot be present at the same time. Nevertheless, it is a requirement that one of them is present.
::::


Multiple providers example:

```yaml
filebeat.inputs:
- type: etw
  id: etw-dnsserver
  enabled: true
  provider.name: Microsoft-Windows-DNSServer
  session_name: DNSServer-Analytical
  trace_level: verbose
  match_any_keyword: 0xffffffffffffffff
  match_all_keyword: 0
- type: etw
  id: etw-security
  enabled: true
  provider.name: Microsoft-Windows-Security-Auditing
  session_name: Security-Auditing
  trace_level: warning
  match_any_keyword: 0xfffffffffffffff
  match_all_keyword: 0
```

## Configuration options [_configuration_options_8]

The `etw` input supports the following configuration options plus the [Common options](#filebeat-input-etw-common-options) described later.


### `file` [_file]

Specifies the path to an .etl file for reading ETW events. This file format is commonly used for storing ETW event logs.


### `provider.guid` [_provider_guid]

Identifies the GUID of an ETW provider. To see available providers, use the command `logman query providers`.


### `provider.name` [_provider_name]

Specifies the name of the ETW provider. Available providers can be listed using `logman query providers`.


### `session_name` [_session_name]

When specifying a provider, a new session is created. This controls the name for the new ETW session it will create. If not specified, the session will be named using the provider ID prefixed by *Elastic-*.


### `trace_level` [_trace_level]

Defines the filtering level for events based on severity. Valid options include critical, error, warning, information, and verbose.


### `match_any_keyword` [_match_any_keyword]

An 8-byte bitmask used for filtering events from specific provider subcomponents based on keyword matching. Any matching keyword will enable the event to be written. Default value is `0xffffffffffffffff` so it matches every available keyword.

Run `logman query providers "<provider.name>"` to list the available keywords for a specific provider.


### `match_all_keyword` [_match_all_keyword]

Similar to MatchAnyKeyword, this 8-byte bitmask filters events that match all specified keyword bits. Default value is `0` to let every event pass.

Run `logman query providers "<provider.name>"` to list the available keywords for a specific provider.


### `session` [_session]

Names an existing ETW session to read from. Existing sessions can be listed using `logman query -ets`.


## Common options [filebeat-input-etw-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_8]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_8]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: etw
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-etw-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: etw
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-etw]

If this option is set to true, the custom [fields](#filebeat-input-etw-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_8]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_8]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_8]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_8]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_8]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


## Metrics [_metrics_7]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs/` path. They can be used to observe the activity of the input.

You must assign a unique `id` to the input to expose metrics.

| Metric | Description |
| --- | --- |
| `session` | Name of the ETW session. |
| `received_events_total` | Total number of events received. |
| `discarded_events_total` | Total number of discarded events. |
| `errors_total` | Total number of errors. |
| `source_lag_time` | Histogram of the difference between timestamped event’s creation and reading. |
| `arrival_period` | Histogram of the elapsed time between event notification callbacks. |
| `processing_time` | Histogram of the elapsed time between event notification callback and publication to the internal queue. |

Histogram metrics are aggregated over the previous 1024 events.


