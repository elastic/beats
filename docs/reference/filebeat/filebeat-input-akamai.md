---
navigation_title: "Akamai"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-akamai.html
applies_to:
  stack: beta
---

# Akamai input [filebeat-input-akamai]

:::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
:::::

Use the `akamai` input to collect security events from the [Akamai SIEM v1 API](https://techdocs.akamai.com/siem-integration/reference/get-security-events) using [EdgeGrid authentication](https://techdocs.akamai.com/developer/docs/authenticate-with-edgegrid).

This input supports:

* EdgeGrid HMAC-SHA256 authentication
* Time-based initial collection with configurable lookback
* Offset-based pagination after the first page
* Cursor-based checkpointing between runs
* Automatic recovery when offsets expire or request timestamps become invalid
* Concurrent worker-based event publishing
* Rate limiting and retry with configurable backoff


## Example configuration [_example_configuration_akamai]

```yaml
filebeat.inputs:
- type: akamai
  resource.url: https://akzz-XXXXXXXX.luna.akamaiapis.net
  config_ids: "12345;67890"

  auth.edgegrid.client_token: "${AKAMAI_CLIENT_TOKEN}"
  auth.edgegrid.client_secret: "${AKAMAI_CLIENT_SECRET}"
  auth.edgegrid.access_token: "${AKAMAI_ACCESS_TOKEN}"

  interval: 1m
  initial_interval: 12h
  recovery_interval: 1h
  event_limit: 10000
  number_of_workers: 3
  invalid_timestamp_retry.max_attempts: 2
```


## Configuration options [_configuration_options_akamai]

The `akamai` input supports the following configuration options plus the [Common options](#filebeat-input-akamai-common-options) described later.


### `resource.url` [_resource_url_akamai]

The base URL for the Akamai API host (for example `https://akzz-XXXXXXXX.luna.akamaiapis.net`). This is used to construct the SIEM API endpoint.

This setting is required.


### `config_ids` [_config_ids_akamai]

A semicolon-separated list of Akamai security configuration IDs to query (for example `"12345;67890"`).

This setting is required.


### `auth.edgegrid.client_token` [_auth_edgegrid_client_token_akamai]

The EdgeGrid client token used for HMAC-SHA256 request signing. This setting is required.


### `auth.edgegrid.client_secret` [_auth_edgegrid_client_secret_akamai]

The EdgeGrid client secret used for HMAC-SHA256 request signing. This setting is required.


### `auth.edgegrid.access_token` [_auth_edgegrid_access_token_akamai]

The EdgeGrid access token used for HMAC-SHA256 request signing. This setting is required.


### `interval` [_interval_akamai]

The polling interval between input cycles. Default: `1m`.


### `initial_interval` [_initial_interval_akamai]

The lookback duration used for the first request when no cursor exists (time-based mode). Cannot exceed `12h` (Akamai API limit). Default: `12h`.


### `recovery_interval` [_recovery_interval_akamai]

The lookback duration used when the input enters recovery mode (for example after an expired offset). Cannot exceed `12h` (Akamai API limit). Default: `1h`.


### `event_limit` [_event_limit_akamai]

The maximum number of events requested per API page. Must be between `1` and `600000`. Default: `10000`.


### `number_of_workers` [_number_of_workers_akamai]

The number of concurrent workers used to publish events from a single fetched page. Must be greater than `0`. Default: `3`.


### `invalid_timestamp_retry.max_attempts` [_invalid_timestamp_retry_max_attempts_akamai]

The number of immediate retries when Akamai responds with HTTP `400` containing `invalid timestamp` (indicating an expired HMAC). Each retry refreshes the HMAC signature before re-sending the request. After retries are exhausted, the input falls back to recovery mode. Default: `2`.


### `resource.timeout` [_resource_timeout_akamai]

Duration before declaring that the HTTP client connection has timed out. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`. Default: `60s`.


### `resource.keep_alive.disable` [_resource_keep_alive_disable_akamai]

This specifies whether to disable keep-alives for HTTP end-points. Default: `true`.


### `resource.keep_alive.max_idle_connections` [_resource_keep_alive_max_idle_connections_akamai]

The maximum number of idle connections across all hosts. Zero means no limit. Default: `0`.


### `resource.keep_alive.max_idle_connections_per_host` [_resource_keep_alive_max_idle_connections_per_host_akamai]

The maximum idle connections to keep per-host. If zero, defaults to two. Default: `0`.


### `resource.keep_alive.idle_connection_timeout` [_resource_keep_alive_idle_connection_timeout_akamai]

The maximum amount of time an idle connection will remain idle before closing itself. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`. Zero means no limit. Default: `0s`.


### `resource.retry.max_attempts` [_resource_retry_max_attempts_akamai]

The maximum number of retries for the HTTP client. Default: `5`.


### `resource.retry.wait_min` [_resource_retry_wait_min_akamai]

The minimum time to wait before a retry is attempted. Default: `1s`.


### `resource.retry.wait_max` [_resource_retry_wait_max_akamai]

The maximum time to wait before a retry is attempted. Default: `60s`.


### `resource.rate_limit.limit` [_resource_rate_limit_limit_akamai]

The value of the maximum overall resource request rate.


### `resource.rate_limit.burst` [_resource_rate_limit_burst_akamai]

The maximum burst size. Burst is the maximum number of resource requests that can be made above the overall rate limit.


### `tracer.enabled` [_tracer_enabled_akamai]

It is possible to log HTTP requests and responses to a local file-system for debugging configurations. This option is enabled by setting `tracer.enabled` to true and setting the `tracer.filename` value. Additional options are available to tune log rotation behavior. To delete existing logs, set `tracer.enabled` to false without unsetting the filename option.

Enabling this option compromises security and should only be used for debugging.


### `tracer.filename` [_tracer_filename_akamai]

To differentiate the trace files generated from different input instances, a placeholder `*` can be added to the filename and will be replaced with the input instance id. For example, `http-request-trace-*.ndjson`.


### `tracer.maxsize` [_tracer_maxsize_akamai]

This value sets the maximum size, in megabytes, the log file will reach before it is rotated. By default logs are allowed to reach 1MB before rotation.


### `tracer.maxage` [_tracer_maxage_akamai]

This specifies the number days to retain rotated log files. If it is not set, log files are retained indefinitely.


### `tracer.maxbackups` [_tracer_maxbackups_akamai]

The number of old logs to retain. If it is not set all old logs are retained subject to the `tracer.maxage` setting.


### `tracer.localtime` [_tracer_localtime_akamai]

Whether to use the host's local time rather that UTC for timestamping rotated log file names.


### `tracer.compress` [_tracer_compress_akamai]

This determines whether rotated logs should be gzip compressed.


## Recovery behavior [_recovery_behavior_akamai]

The input uses cursor-aware recovery logic to handle Akamai API error responses:

| Scenario | Behavior |
| --- | --- |
| `416` (offset expired) | Drops the cursor and switches to time-based recovery using `recovery_interval` as the lookback window. |
| `400` with `invalid timestamp` | Refreshes the HMAC signature and retries immediately, up to `invalid_timestamp_retry.max_attempts` times. |
| Invalid timestamp retries exhausted | Falls back to the same cursor-drop recovery path as `416`. |
| Other `400` responses | Treated as non-recoverable for the current poll cycle. Logged and the input moves to the next polling interval. |


## Metrics [_metrics_akamai]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `resource` | URL of the input resource. |
| `akamai_requests_total` | Total number of API requests made. |
| `akamai_requests_success_total` | Total number of successful API requests. |
| `akamai_requests_errors_total` | Total number of failed API requests. |
| `batches_received_total` | Number of event batches received from the API. |
| `batches_published_total` | Number of event batches successfully published. |
| `events_received_total` | Total number of events received. |
| `events_published_total` | Total number of events published. |
| `events_publish_failed_total` | Total number of individual event publish failures. |
| `errors_total` | Total number of errors encountered. |
| `recovery_mode_entries_total` | Number of times recovery mode was entered (offset expired or HMAC retries exhausted). |
| `offset_expired_total` | Number of `416` offset-expired responses received. |
| `hmac_refresh_total` | Number of HMAC signature refreshes triggered by invalid timestamp errors. |
| `api_400_fatal_total` | Number of non-recoverable `400` responses received. |
| `cursor_drops_total` | Number of times the cursor was dropped and reset. |
| `workers_active_gauge` | Number of currently active event-publishing workers. |
| `worker_utilization` | Worker utilization ratio (`0`--`1`), updated every 5 seconds. |
| `request_processing_time` | Histogram of request processing times in nanoseconds. |
| `batch_processing_time` | Histogram of batch processing times in nanoseconds (receipt to ACK). |
| `events_per_batch` | Histogram of the number of events per batch. |
| `failed_events_per_page` | Histogram of the number of failed event publishes per page. |
| `response_latency` | Histogram of API response latencies in nanoseconds. |


## Common options [filebeat-input-akamai-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_akamai]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_akamai]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: akamai
  . . .
  tags: ["akamai", "security"]
```


#### `fields` [filebeat-input-akamai-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: akamai
  . . .
  fields:
    env: production
```


#### `fields_under_root` [fields-under-root-akamai]

If this option is set to true, the custom [fields](#filebeat-input-akamai-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_akamai]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_akamai]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_akamai]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_akamai]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event's metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_akamai]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.
