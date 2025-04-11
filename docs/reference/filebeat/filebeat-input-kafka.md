---
navigation_title: "Kafka"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-kafka.html
---

# Kafka input [filebeat-input-kafka]


Use the `kafka` input to read from topics in a Kafka cluster.

To configure this input, specify a list of one or more [`hosts`](/reference/filebeat/logstash-output.md#hosts) in the cluster to bootstrap the connection with, a list of [`topics`](#topics) to track, and a [`group_id`](#groupid) for the connection.

Example configuration:

```yaml
filebeat.inputs:
- type: kafka
  hosts:
    - kafka-broker-1:9092
    - kafka-broker-2:9092
  topics: ["my-topic"]
  group_id: "filebeat"
```

The following example shows how to use the `kafka` input to ingest data from Microsoft Azure Event Hubs that have Kafka compatibility enabled:

```yaml
filebeat.inputs:
- type: kafka
  hosts: ["<your event hub namespace>.servicebus.windows.net:9093"]
  topics: ["<your event hub instance>"]
  group_id: "<your consumer group>"

  username: "$ConnectionString"
  password: "<your connection string>"
  ssl.enabled: true
```

For more details on the mapping between Kafka and Event Hubs configuration parameters, see the [Azure documentation](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-for-kafka-ecosystem-overview).

## Compatibility [kafka-input-compatibility]

This input works with all Kafka versions in between 0.11 and 2.8.0. Older versions might work as well, but are not supported.


## Configuration options [filebeat-input-kafka-options]

The `kafka` input supports the following configuration options plus the [Common options](#filebeat-input-kafka-common-options) described later.

::::{note}
If you’re using {{agent}} with a Kafka input and need to increase throughput, we recommend scaling horizontally by additional {{agents}} to read from the Kafka topic. Note that each {{agent}} reads concurrently from each of the partitions it has been assigned.
::::



#### `hosts` [kafka-hosts]

A list of Kafka bootstrapping hosts (brokers) for this cluster.


#### `topics` [topics]

A list of topics to read from.


#### `group_id` [groupid]

The Kafka consumer group id.


#### `client_id` [_client_id_3]

The Kafka client id (optional).


#### `version` [_version_2]

The version of the Kafka protocol to use (defaults to `"2.1.0"`). When using Kafka 4.0 and newer, the version must be set to at least `"2.1.0"`.


#### `initial_offset` [_initial_offset]

The initial offset to start reading, either "oldest" or "newest". Defaults to "oldest".

### `connect_backoff` [_connect_backoff]

How long to wait before trying to reconnect to the kafka cluster after a fatal error. Default is 30s.


### `consume_backoff` [_consume_backoff]

How long to wait before retrying a failed read. Default is 2s.


### `max_wait_time` [_max_wait_time]

How long to wait for the minimum number of input bytes while reading. Default is 250ms.


### `wait_close` [_wait_close]

When shutting down, how long to wait for in-flight messages to be delivered and acknowledged.


### `isolation_level` [_isolation_level]

This configures the Kafka group isolation level:

* `"read_uncommitted"` returns *all* messages in the message channel.
* `"read_committed"` hides messages that are part of an aborted transaction.

The default is `"read_uncommitted"`.


### `fetch` [_fetch]

Kafka fetch settings:

**`min`**
:   The minimum number of bytes to wait for. Defaults to 1.

**`default`**
:   The default number of bytes to read per request. Defaults to 1MB.

**`max`**
:   The maximum number of bytes to read per request. Defaults to 0 (no limit).


### `expand_event_list_from_field` [_expand_event_list_from_field_2]

If the fileset using this input expects to receive multiple messages bundled under a specific field then the config option `expand_event_list_from_field` value can be assigned the name of the field. For example in the case of azure filesets the events are found under the json object "records".

```json
{
"records": [ {event1}, {event2}]
}
```

This setting will be able to split the messages under the group value (*records*) into separate events.


### `rebalance` [_rebalance]

Kafka rebalance settings:

**`strategy`**
:   Either `"range"` or `"roundrobin"`. Defaults to `"range"`.

**`timeout`**
:   How long to wait for an attempted rebalance. Defaults to 60s.

**`max_retries`**
:   How many times to retry if rebalancing fails. Defaults to 4.

**`retry_backoff`**
:   How long to wait after an unsuccessful rebalance attempt. Defaults to 2s.


### `sasl.mechanism` [_sasl_mechanism]

The SASL mechanism to use when connecting to Kafka. It can be one of:

* `PLAIN` for SASL/PLAIN.
* `SCRAM-SHA-256` for SCRAM-SHA-256.
* `SCRAM-SHA-512` for SCRAM-SHA-512.

If `sasl.mechanism` is not set, `PLAIN` is used if `username` and `password` are provided. Otherwise, SASL authentication is disabled.

To use `GSSAPI` mechanism to authenticate with Kerberos, you must leave this field empty, and use the [`kerberos`](/reference/filebeat/kafka-output.md#kerberos-option-kafka) options.


### `kerberos` [_kerberos]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Configuration options for Kerberos authentication.

See [Kerberos](/reference/filebeat/configuration-kerberos.md) for more information.


#### `parsers` [_parsers_3]

This option expects a list of parsers that the payload has to go through.

Available parsers:

* `ndjson`
* `multiline`


#### `ndjson` [_ndjson]

These options make it possible for Filebeat to decode the payload as JSON messages.

Example configuration:

```yaml
- ndjson:
  target: ""
  add_error_key: true
  message_key: log
```

**`target`**
:   The name of the new JSON object that should contain the parsed key value pairs. If you leave it empty, the new keys will go under root.

**`overwrite_keys`**
:   Values from the decoded JSON object overwrite the fields that Filebeat normally adds (type, source, offset, etc.) in case of conflicts. Disable it if you want to keep previously added values.

**`expand_keys`**
:   If this setting is enabled, Filebeat will recursively de-dot keys in the decoded JSON, and expand them into a hierarchical object structure. For example, `{"a.b.c": 123}` would be expanded into `{"a":{"b":{"c":123}}}`. This setting should be enabled when the input is produced by an [ECS logger](https://github.com/elastic/ecs-logging).

**`add_error_key`**
:   If this setting is enabled, Filebeat adds an "error.message" and "error.type: json" key in case of JSON unmarshalling errors or when a `message_key` is defined in the configuration but cannot be used.

**`message_key`**
:   An optional configuration setting that specifies a JSON key on which to apply the line filtering and multiline settings. If specified the key must be at the top level in the JSON object and the value associated with the key must be a string, otherwise no filtering or multiline aggregation will occur.

**`document_id`**
:   Option configuration setting that specifies the JSON key to set the document id. If configured, the field will be removed from the original JSON document and stored in `@metadata._id`

**`ignore_decoding_error`**
:   An optional configuration setting that specifies if JSON decoding errors should be logged or not. If set to true, errors will not be logged. The default is false.


#### `multiline` [_multiline_5]

Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.



## Common options [filebeat-input-kafka-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_15]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_15]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: kafka
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-kafka-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: kafka
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-kafka]

If this option is set to true, the custom [fields](#filebeat-input-kafka-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_15]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_15]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_15]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_15]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_15]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


