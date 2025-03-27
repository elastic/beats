---
navigation_title: "NetFlow"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-netflow.html
---

# NetFlow input [filebeat-input-netflow]


Use the `netflow` input to read NetFlow and IPFIX exported flows and options records over UDP.

This input supports NetFlow versions 1, 5, 6, 7, 8 and 9, as well as IPFIX. For NetFlow versions older than 9, fields are mapped automatically to NetFlow v9.

Example configuration:

```yaml
filebeat.inputs:
- type: netflow
  max_message_size: 10KiB
  host: "0.0.0.0:2055"
  protocols: [ v5, v9, ipfix ]
  expiration_timeout: 30m
  queue_size: 8192
  custom_definitions:
  - path/to/fields.yml
  detect_sequence_reset: true
```

## Configuration options [_configuration_options_13]

The `netflow` input supports the following configuration options plus the [Common options](#filebeat-input-netflow-common-options) described later.


### `max_message_size` [filebeat-input-netflow-udp-max-message-size]

The maximum size of the message received over UDP. The default is `10KiB`.


### `host` [filebeat-input-netflow-udp-host]

The host and UDP port to listen on for event streams.


### `network` [filebeat-input-netflow-udp-network]

The network type. Acceptable values are: "udp" (default), "udp4", "udp6"


### `read_buffer` [filebeat-input-netflow-udp-read-buffer]

The size of the read buffer on the UDP socket. If not specified the default from the operating system will be used.


### `timeout` [filebeat-input-netflow-udp-timeout]

The read and write timeout for socket operations. The default is `5m`.


### `protocols` [protocols]

List of enabled protocols. Valid values are `v1`, `v5`, `v6`, `v7`, `v8`, `v9` and `ipfix`.


### `expiration_timeout` [expiration_timeout]

The time before an idle session or unused template is expired. Only applicable to v9 and IPFIX protocols. A value of zero disables expiration.


### `share_templates` [share_templates]

This option allows v9 and ipfix templates to be shared within a session without reference to the origin of the template.

Note that setting this to true is not recommended as it can result in the wrong template being applied under certain conditions, but it may be required for some systems.


### `queue_size` [queue_size]

The maximum number of packets that can be queued for processing. Use this setting to avoid packet-loss when dealing with occasional bursts of traffic.


### `workers` [workers]

The number of workers to read and decode concurrently netflow packets. Default is `1`. Note that in order to maximize the performance gains of multiple workers it is advised to switch the output to `throughput` preset ([link](/reference/filebeat/elasticsearch-output.md#_preset)).


### `custom_definitions` [custom_definitions]

A list of paths to field definitions YAML files. These allow to update the NetFlow/IPFIX fields with vendor extensions and to override existing fields.

The expected format is the same as used by Logstash’s NetFlow codec [ipfix_definitions](logstash-docs-md://lsr/plugins-codecs-netflow.md#plugins-codecs-netflow-ipfix_definitions) and [netflow_definitions](logstash-docs-md://lsr/plugins-codecs-netflow.md#plugins-codecs-netflow-netflow_definitions). Filebeat will detect which of the two formats is used.

NetFlow format example:

```yaml
id:
- default length in bytes
- :name
id:
- :uintN or :intN: or :ip4_addr or :ip6_addr or :mac_addr or :string
- :name
id:
- :skip
```

Where `id` is the numeric field ID.

The IPFIX format similar, but grouped by Private Enterprise Number (PEN):

```yaml
pen1:
  id:
  - :uintN or :ip4_addr or :ip6_addr or :mac_addr or :string
  - :name
  id:
  - :skip
pen2:
  id:
  - :octetarray
  - :name
```

Note that fields are shared between NetFlow V9 and IPFIX. Changes to IPFIX PEN zero are equivalent to changes to NetFlow fields.

::::{warning}
Overriding the names and/or types of standard fields can prevent mapping of ECS fields to function properly.
::::



### `detect_sequence_reset` [detect_sequence_reset]

Flag controlling whether Filebeat should monitor sequence numbers in the Netflow packets to detect an Exporting Process reset. When this condition is detected, record templates for the given exporter will be dropped. This will cause flow loss until the exporter provides new templates. If set to `false`, Filebeat will ignore sequence numbers, which can cause some invalid flows if the exporter process is reset. This option is only applicable to Netflow V9 and IPFIX. Default is `true`.


### `internal_networks` [internal_networks]

A list of CIDR ranges describing the IP addresses that you consider internal. This is used in determining the values of `source.locality`, `destination.locality`, and `flow.locality`. The values can be either a CIDR value or one of the named ranges supported by the [`network`](/reference/filebeat/defining-processors.md#condition-network) condition. The default value is `[private]` which classifies RFC 1918 (IPv4) and RFC 4193 (IPv6) addresses as internal.


## Common options [filebeat-input-netflow-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_18]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_18]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: netflow
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-netflow-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: netflow
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-netflow]

If this option is set to true, the custom [fields](#filebeat-input-netflow-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_18]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_18]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_18]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_18]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_18]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


## Metrics [_metrics_13]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs/` path. They can be used to observe the activity of the input.

You must assign a unique `id` to the input to expose metrics.

| Metric | Description |
| --- | --- |
| `device` | Host/port of the UDP stream. |
| `udp_read_buffer_length_gauge` | Size of the UDP socket buffer length in bytes (gauge). |
| `received_events_total` | Total number of packets (events) that have been received. |
| `received_bytes_total` | Total number of bytes received. |
| `receive_queue_length` | Aggregated size of the system receive queues (IPv4 and IPv6) (linux only) (gauge). |
| `system_packet_drops` | Aggregated number of system packet drops (IPv4 and IPv6) (linux only) (gauge). |
| `arrival_period` | Histogram of the time between successive packets in nanoseconds. |
| `processing_time` | Histogram of the time taken to process packets in nanoseconds. |
| `discarded_events_total` | Total number of discarded events. |
| `decode_errors_total` | Total number of errors at decoding a packet. |
| `flows_total` | Total number of received flows. |
| `open_connections` | Number of current active netflow sessions. |

Histogram metrics are aggregated over the previous 1024 events.


