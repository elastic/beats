---
navigation_title: "Kafka"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/kafka-output.html
---

# Configure the Kafka output [kafka-output]


The Kafka output sends events to Apache Kafka.

To use this output, edit the Auditbeat configuration file to disable the {{es}} output by commenting it out, and enable the Kafka output by uncommenting the Kafka section.

::::{note}
For Kafka version 0.10.0.0+ the message creation timestamp is set by beats and equals to the initial timestamp of the event. This affects the retention policy in Kafka: for example, if a beat event was created 2 weeks ago, the retention policy is set to 7 days and the message from beats arrives to Kafka today, it’s going to be immediately discarded since the timestamp value is before the last 7 days. It’s possible to change this behavior by setting timestamps on message arrival instead, so the message is not discarded but kept for 7 more days. To do that, please set `log.message.timestamp.type` to `LogAppendTime` (default `CreateTime`) in the Kafka configuration.
::::


Example configuration:

```yaml
output.kafka:
  # initial brokers for reading cluster metadata
  hosts: ["kafka1:9092", "kafka2:9092", "kafka3:9092"]

  # message topic selection + partitioning
  topic: '%{[fields.log_topic]}'
  partition.round_robin:
    reachable_only: false

  required_acks: 1
  compression: gzip
  max_message_bytes: 1000000
```

::::{note}
Events bigger than [`max_message_bytes`](#kafka-max_message_bytes) will be dropped. To avoid this problem, make sure Auditbeat does not generate events bigger than [`max_message_bytes`](#kafka-max_message_bytes).
::::


## Compatibility [kafka-compatibility]

This output can connect to Kafka version 0.8.2.0 and later. Older versions might work as well, but are not supported. When using Kafka 4.0 and newer, the version must be set to at least `"2.1.0"`


## Configuration options [_configuration_options_4]

You can specify the following options in the `kafka` section of the `auditbeat.yml` config file:

### `enabled` [_enabled_3]

The `enabled` config is a boolean setting to enable or disable the output. If set to false, the output is disabled.

The default value is `true`.


### `hosts` [_hosts]

The list of Kafka broker addresses from where to fetch the cluster metadata. The cluster metadata contain the actual Kafka brokers events are published to.


### `version` [_version]

Kafka protocol version that Auditbeat will request when connecting. Defaults to 2.1.0. When using Kafka 4.0 and newer, the version must be set to at least `"2.1.0"`

Valid values are all kafka releases in between `0.8.2.0` and `2.6.0`.

The protocol version controls the Kafka client features available to Auditbeat; it does not prevent Auditbeat from connecting to Kafka versions newer than the protocol version.

See [Compatibility](#kafka-compatibility) for information on supported versions.


### `username` [_username_2]

The username for connecting to Kafka. If username is configured, the password must be configured as well.


### `password` [_password_2]

The password for connecting to Kafka.


### `sasl.mechanism` [_sasl_mechanism]

The SASL mechanism to use when connecting to Kafka. It can be one of:

* `PLAIN` for SASL/PLAIN.
* `SCRAM-SHA-256` for SCRAM-SHA-256.
* `SCRAM-SHA-512` for SCRAM-SHA-512.

If `sasl.mechanism` is not set, `PLAIN` is used if `username` and `password` are provided. Otherwise, SASL authentication is disabled.

To use `GSSAPI` mechanism to authenticate with Kerberos, you must leave this field empty, and use the [`kerberos`](#kerberos-option-kafka) options.


### `topic` [topic-option-kafka]

The Kafka topic used for produced events.

You can set the topic dynamically by using a format string to access any event field. For example, this configuration uses a custom field, `fields.log_topic`, to set the topic for each event:

```yaml
topic: '%{[fields.log_topic]}'
```

::::{tip}
To learn how to add custom fields to events, see the [`fields`](/reference/auditbeat/configuration-general-options.md#libbeat-configuration-fields) option.
::::


See the [`topics`](#topics-option-kafka) setting for other ways to set the topic dynamically.


### `topics` [topics-option-kafka]

An array of topic selector rules. Each rule specifies the `topic` to use for events that match the rule. During publishing, Auditbeat sets the `topic` for each event based on the first matching rule in the array. Rules can contain conditionals, format string-based fields, and name mappings. If the `topics` setting is missing or no rule matches, the [`topic`](#topic-option-kafka) field is used.

Rule settings:

**`topic`**
:   The topic format string to use.  If this string contains field references, such as `%{[fields.name]}`, the fields must exist, or the rule fails.

**`mappings`**
:   A dictionary that takes the value returned by `topic` and maps it to a new name.

**`default`**
:   The default string value to use if `mappings` does not find a match.

**`when`**
:   A condition that must succeed in order to execute the current rule. All the [conditions](/reference/auditbeat/defining-processors.md#conditions) supported by processors are also supported here.

The following example sets the topic based on whether the message field contains the specified string:

```yaml
output.kafka:
  hosts: ["localhost:9092"]
  topic: "logs-%{[agent.version]}"
  topics:
    - topic: "critical-%{[agent.version]}"
      when.contains:
        message: "CRITICAL"
    - topic: "error-%{[agent.version]}"
      when.contains:
        message: "ERR"
```

This configuration results in topics named `critical-[version]`, `error-[version]`, and `logs-[version]`.


### `key` [_key]

Optional formatted string specifying the Kafka event key. If configured, the event key can be extracted from the event using a format string.

See the Kafka documentation for the implications of a particular choice of key; by default, the key is chosen by the Kafka cluster.


### `partition` [_partition]

Kafka output broker event partitioning strategy. Must be one of `random`, `round_robin`, or `hash`. By default the `hash` partitioner is used.

**`random.group_events`**
:   Sets the number of events to be published to the same partition, before the partitioner selects a new partition by random. The default value is 1 meaning after each event a new partition is picked randomly.

**`round_robin.group_events`**
:   Sets the number of events to be published to the same partition, before the partitioner selects the next partition. The default value is 1 meaning after each event the next partition will be selected.

**`hash.hash`**
:   List of fields used to compute the partitioning hash value from. If no field is configured, the events `key` value will be used.

**`hash.random`**
:   Randomly distribute events if no hash or key value can be computed.

All partitioners will try to publish events to all partitions by default. If a partition’s leader becomes unreachable for the beat, the output might block. All partitioners support setting `reachable_only` to overwrite this behavior. If `reachable_only` is set to `true`, events will be published to available partitions only.

::::{note}
Publishing to a subset of available partitions potentially increases resource usage because events may become unevenly distributed.
::::



### `headers` [_headers_2]

A header is a key-value pair, and multiple headers can be included with the same `key`. Only string values are supported. These headers will be included in each produced Kafka message.

```yaml
output.kafka:
  hosts: ["localhost:9092"]
  topic: "logs-%{[agent.version]}"
  headers:
    - key: "some-key"
      value: "some value"
    - key: "another-key"
      value: "another value"
```


### `client_id` [_client_id]

The configurable ClientID used for logging, debugging, and auditing purposes. The default is "beats".


### `codec` [_codec]

Output codec configuration. If the `codec` section is missing, events will be json encoded.

See [Change the output codec](/reference/auditbeat/configuration-output-codec.md) for more information.


### `metadata` [_metadata]

Kafka metadata update settings. The metadata do contain information about brokers, topics, partition, and active leaders to use for publishing.

**`refresh_frequency`**
:   Metadata refresh interval. Defaults to 10 minutes.

**`full`**
:   Strategy to use when fetching metadata, when this option is `true`, the client will maintain a full set of metadata for all the available topics, if the this option is set to `false` it will only refresh the metadata for the configured topics. The default is false.

**`retry.max`**
:   Total number of metadata update retries when cluster is in middle of leader election. The default is 3.

**`retry.backoff`**
:   Waiting time between retries during leader elections. Default is 250ms.


### `max_retries` [_max_retries_3]

The number of times to retry publishing an event after a publishing failure. After the specified number of retries, the events are typically dropped.

Set `max_retries` to a value less than 0 to retry until all events are published.

The default is 3.


### `backoff.init` [_backoff_init_2]

The number of seconds to wait before trying to republish to Kafka after a network error. After waiting `backoff.init` seconds, Auditbeat tries to republish. If the attempt fails, the backoff timer is increased exponentially up to `backoff.max`. After a successful publish, the backoff timer is reset. The default is 1s.


### `backoff.max` [_backoff_max_2]

The maximum number of seconds to wait before attempting to republish to Kafka after a network error. The default is 60s.


### `bulk_max_size` [_bulk_max_size_2]

The maximum number of events to bulk in a single Kafka request. The default is 2048.


### `bulk_flush_frequency` [_bulk_flush_frequency]

Duration to wait before sending bulk Kafka request. 0 is no delay. The default is 0.


### `timeout` [_timeout_3]

The number of seconds to wait for responses from the Kafka brokers before timing out. The default is 30 (seconds).


### `broker_timeout` [_broker_timeout]

The maximum duration a broker will wait for number of required ACKs. The default is 10s.


### `channel_buffer_size` [_channel_buffer_size]

Per Kafka broker number of messages buffered in output pipeline. The default is 256.


### `keep_alive` [_keep_alive]

The keep-alive period for an active network connection. If 0s, keep-alives are disabled. The default is 0 seconds.


### `compression` [_compression]

Sets the output compression codec. Must be one of `none`, `snappy`, `lz4`, `gzip` and `zstd`. The default is `gzip`.

::::{admonition} Known issue with Azure Event Hub for Kafka
:class: important

When targeting Azure Event Hub for Kafka, set `compression` to `none` as the provided codecs are not supported.

::::



### `compression_level` [_compression_level_2]

Sets the compression level used by gzip. Setting this value to 0 disables compression. The compression level must be in the range of 1 (best speed) to 9 (best compression).

Increasing the compression level will reduce the network usage but will increase the cpu usage.

The default value is 4.


### `max_message_bytes` [kafka-max_message_bytes]

The maximum permitted size of JSON-encoded messages. Bigger messages will be dropped. The default value is 1000000 (bytes). This value should be equal to or less than the broker’s `message.max.bytes`.


### `required_acks` [_required_acks]

The ACK reliability level required from broker. 0=no response, 1=wait for local commit, -1=wait for all replicas to commit. The default is 1.

Note: If set to 0, no ACKs are returned by Kafka. Messages might be lost silently on error.


### `ssl` [_ssl_3]

Configuration options for SSL parameters like the root CA for Kafka connections. The Kafka host keystore should be created with the `-keyalg RSA` argument to ensure it uses a cipher supported by [Filebeat’s Kafka library](https://github.com/Shopify/sarama/wiki/Frequently-Asked-Questions#why-cant-sarama-connect-to-my-kafka-cluster-using-ssl). See [SSL](/reference/auditbeat/configuration-ssl.md) for more information.


### `kerberos` [kerberos-option-kafka]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Configuration options for Kerberos authentication.

See [Kerberos](/reference/auditbeat/configuration-kerberos.md) for more information.


### `queue` [_queue_3]

Configuration options for internal queue.

See [Internal queue](/reference/auditbeat/configuring-internal-queue.md) for more information.

Note:`queue` options can be set under `auditbeat.yml` or the `output` section but not both.



