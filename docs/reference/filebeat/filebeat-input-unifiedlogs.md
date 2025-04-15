---
navigation_title: "Unified Logs"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-unifiedlogs.html
  # That link will 404 until 8.18 is current
  # (see https://www.elastic.co/guide/en/beats/filebeat/8.18/filebeat-input-unifiedlogs.html)
---

# Unified Logs input [filebeat-input-unifiedlogs]


::::{note}
Only available for MacOS.
::::


The unified logging system provides a comprehensive and performant API to capture telemetry across all levels of the system. This system centralizes the storage of log data in memory and on disk, rather than writing that data to a text-based log file.

The input interacts with the `log` command-line tool to provide access to the events.

The input starts streaming events from the current point in time unless a start date or the `backfill` options are set. When restarted it will continue where it left off.

Alternatively, it can also do one off operations, such as:

* Stream events contained in a `.logarchive` file.
* Stream events contained in a `.tracev3` file.
* Stream events in a specific time span, by providing a specific end date.

After this one off operations complete, the input will stop.

Other configuration options can be specified to filter what events to process.

::::{note}
The input can cause some duplicated events when backfilling and/or restarting. This is caused by how the underlying fetching method works and should be taken into account when using the input.
::::


Example configuration:

Process all old and new logs:

```yaml
filebeat.inputs:
- type: unifiedlogs
  id: unifiedlogs-id
  enabled: true
  backfill: true
```

Process logs with predicate filters:

```yaml
filebeat.inputs:
- type: unifiedlogs
  id: unifiedlogs-id
  enabled: true
  predicate:
    # Captures keychain.db unlock events
    - 'process == "loginwindow" && sender == "Security"'
    # Captures user login events
    - 'process == "logind"'
    # Captures command line activity run with elevated privileges
    - 'process == "sudo"'
```

## Configuration options [_configuration_options_21]

The `unifiedlogs` input supports the following configuration options plus the [Common options](filebeat-input-unifiedlogs.md#filebeat-input-unifiedlogs-common-options) described later.


### `archive_file` [_archive_file]

Display events stored in the given archive. The archive must be a valid log archive bundle with the suffix `.logarchive`.


### `trace_file` [_trace_file]

Display events stored in the given `.tracev3` file. In order to be decoded, the file must be contained within a valid `.logarchive`


### `start` [_start]

Shows content starting from the provided date. The following date/time formats are accepted: `YYYY-MM-DD`, `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD HH:MM:SSZZZZZ`.


### `end` [_end]

Shows content up to the provided date. The following date/time formats are accepted: `YYYY-MM-DD`, `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD HH:MM:SSZZZZZ`.


### `predicate` [_predicate]

Filters messages using the provided predicate based on NSPredicate. A compound predicate or multiple predicates can be provided as a list.

For detailed information on the use of predicate based filtering, please refer to the [Predicate Programming Guide](https://developer.apple.com/library/mac/documentation/Cocoa/Conceptual/Predicates/Articles/pSyntax.html).


### `process` [_process]

A list of the processes on which to operate. It accepts a PID or process name.


### `source` [_source]

Include symbol names and source line numbers for messages, if available. Default: `false`.


### `info` [_info]

Disable or enable info level messages. Default: `false`.


### `debug` [_debug]

Disable or enable debug level messages. Default: `false`.


### `backtrace` [_backtrace]

Disable or enable display of backtraces. Default: `false`.


### `signpost` [_signpost]

Disable or enable display of signposts. Default: `false`.


### `unreliable` [_unreliable]

Annotate events with whether the log was emitted unreliably. Default: `false`.


### `mach_continuous_time` [_mach_continuous_time]

Use mach continuous time timestamps rather than walltime. Default: `false`.


### `backfill` [_backfill]

If set to true the input will process all available logs since the beginning of time the first time it starts. Default: `false`.


## Common options [filebeat-input-unifiedlogs-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_28]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_27]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: unifiedlogs
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-unifiedlogs-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: unifiedlogs
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-unifiedlogs]

If this option is set to true, the custom [fields](filebeat-input-unifiedlogs.md#filebeat-input-unifiedlogs-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_27]

A list of processors to apply to the input data.

See [Processors](filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_27]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_27]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_27]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_27]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


## Metrics [_metrics_17]

This input exposes metrics under the [HTTP monitoring endpoint](http-endpoint.md). These metrics are exposed under the `/inputs/` path. They can be used to observe the activity of the input.

You must assign a unique `id` to the input to expose metrics.

| Metric | Description |
| --- | --- |
| `errors_total` | Total number of errors. |


