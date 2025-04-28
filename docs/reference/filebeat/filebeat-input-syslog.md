---
navigation_title: "Syslog"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-syslog.html
---

# Syslog input [filebeat-input-syslog]

:::{admonition} Deprecated in 8.14.0
The syslog input is deprecated. Please use the [`syslog`](/reference/filebeat/syslog.md) processor for processing syslog messages.
:::

The `syslog` input reads Syslog events as specified by RFC 3164 and RFC 5424, over TCP, UDP, or a Unix stream socket.

Example configurations:

```yaml
filebeat.inputs:
- type: syslog
  format: rfc3164
  protocol.udp:
    host: "localhost:9000"
```

```yaml
filebeat.inputs:
- type: syslog
  format: rfc5424
  protocol.tcp:
    host: "localhost:9000"
```

```yaml
filebeat.inputs:
- type: syslog
  format: auto
  protocol.unix:
    path: "/path/to/syslog.sock"
```

## Configuration options [_configuration_options_18]

The `syslog` input configuration includes format, protocol specific options, and the [Common options](#filebeat-input-syslog-common-options) described later..

### `format` [_format_2]

The syslog variant to use, `rfc3164` or `rfc5424`. To automatically detect the format from the log entries, set this option to `auto`. The default is `rfc3164`.


### `timezone` [_timezone]

IANA time zone name (e.g. `America/New_York`) or fixed time offset (e.g. `+0200`) to use when parsing syslog timestamps that do not contain a time zone. `Local` may be specified to use the machine’s local time zone. Defaults to `Local`.


### Protocol `udp`: [_protocol_udp]


### `max_message_size` [filebeat-input-syslog-udp-max-message-size]

The maximum size of the message received over UDP. The default is `10KiB`.


### `host` [filebeat-input-syslog-udp-host]

The host and UDP port to listen on for event streams.


### `network` [filebeat-input-syslog-udp-network]

The network type. Acceptable values are: "udp" (default), "udp4", "udp6"


### `read_buffer` [filebeat-input-syslog-udp-read-buffer]

The size of the read buffer on the UDP socket. If not specified the default from the operating system will be used.


### `timeout` [filebeat-input-syslog-udp-timeout]

The read and write timeout for socket operations. The default is `5m`.


### Protocol `tcp`: [_protocol_tcp]


### `max_message_size` [filebeat-input-syslog-tcp-max-message-size]

The maximum size of the message received over TCP. The default is `20MiB`.


### `host` [filebeat-input-syslog-tcp-host]

The host and TCP port to listen on for event streams.


### `network` [filebeat-input-syslog-tcp-network]

The network type. Acceptable values are: "tcp" (default), "tcp4", "tcp6"


### `framing` [filebeat-input-syslog-tcp-framing]

Specify the framing used to split incoming events.  Can be one of `delimiter` or `rfc6587`.  `delimiter` uses the characters specified in `line_delimiter` to split the incoming events.  `rfc6587` supports octet counting and non-transparent framing as described in [RFC6587](https://tools.ietf.org/html/rfc6587).  `line_delimiter` is used to split the events in non-transparent framing.  The default is `delimiter`.


### `line_delimiter` [filebeat-input-syslog-tcp-line-delimiter]

Specify the characters used to split the incoming events. The default is *\n*.


### `max_connections` [filebeat-input-syslog-tcp-max-connections]

The at most number of connections to accept at any given point in time.


### `timeout` [filebeat-input-syslog-tcp-timeout]

The number of seconds of inactivity before a remote connection is closed. The default is `300s`.


#### `ssl` [filebeat-input-syslog-tcp-ssl]

Configuration options for SSL parameters like the certificate, key and the certificate authorities to use.

See [SSL](/reference/filebeat/configuration-ssl.md) for more information.


### Protocol `unix`: [_protocol_unix]


### `max_message_size` [filebeat-input-syslog-unix-max-message-size]

The maximum size of the message received over the socket. The default is `20MiB`.


### `path` [filebeat-input-syslog-unix-path]

The path to the Unix socket that will receive events.


### `socket_type` [filebeat-input-syslog-unix-socket-type]

The type to of the Unix socket that will receive events. Valid values are `stream` and `datagram`. The default is `stream`.


### `group` [filebeat-input-syslog-unix-group]

The group ownership of the Unix socket that will be created by Filebeat. The default is the primary group name for the user Filebeat is running as. This option is ignored on Windows.


### `mode` [filebeat-input-syslog-unix-mode]

The file mode of the Unix socket that will be created by Filebeat. This is expected to be a file mode as an octal string. The default value is the system default (generally `0755`).


### `framing` [filebeat-input-syslog-unix-framing]

Specify the framing used to split incoming events.  Can be one of `delimiter` or `rfc6587`.  `delimiter` uses the characters specified in `line_delimiter` to split the incoming events.  `rfc6587` supports octet counting and non-transparent framing as described in [RFC6587](https://tools.ietf.org/html/rfc6587).  `line_delimiter` is used to split the events in non-transparent framing.  The default is `delimiter`.


### `line_delimiter` [filebeat-input-syslog-unix-line-delimiter]

Specify the characters used to split the incoming events. The default is *\n*.


### `max_connections` [filebeat-input-syslog-unix-max-connections]

The at most number of connections to accept at any given point in time.


### `timeout` [filebeat-input-syslog-unix-timeout]

The number of seconds of inactivity before a connection is closed. The default is `300s`.

See [SSL](/reference/filebeat/configuration-ssl.md) for more information.



## Common options [filebeat-input-syslog-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_25]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_24]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: syslog
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-syslog-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: syslog
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-syslog]

If this option is set to true, the custom [fields](#filebeat-input-syslog-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_24]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_24]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_24]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_24]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_24]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


