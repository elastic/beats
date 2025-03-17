---
navigation_title: "GCP Pub/Sub"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-gcp-pubsub.html
---

# GCP Pub/Sub input [filebeat-input-gcp-pubsub]


Use the `gcp-pubsub` input to read messages from a Google Cloud Pub/Sub topic subscription.

This input can, for example, be used to receive Stackdriver logs that have been exported to a Google Cloud Pub/Sub topic.

Multiple Filebeat instances can be configured to read from the same subscription to achieve high-availability or increased throughput.

Example configuration:

```yaml
filebeat.inputs:
- type: gcp-pubsub
  project_id: my-gcp-project-id
  topic: vpc-firewall-logs-topic
  subscription.name: filebeat-vpc-firewall-logs-sub
  credentials_file: ${path.config}/my-pubsub-subscriber-credentials.json
```

## Configuration options [_configuration_options_9]

The `gcp-pubsub` input supports the following configuration options plus the [Common options](#filebeat-input-gcp-pubsub-common-options) described later.


### `project_id` [_project_id]

Google Cloud project ID. Required.


### `topic` [_topic]

Google Cloud Pub/Sub topic name. Required.


### `subscription.name` [_subscription_name]

Name of the subscription to read from. Required.


### `subscription.create` [_subscription_create]

Boolean value that configures the input to create the subscription if it does not exist. The default value is `true`.


### `subscription.num_goroutines` [_subscription_num_goroutines]

Number of goroutines to create to read from the subscription. This does not limit the number of messages that can be processed concurrently or the maximum number of goroutines the input will create. Even with one goroutine, many messages might be processed at once, because that goroutine may continually receive messages. To limit the number of messages being processed concurrently, set `subscription.max_outstanding_messages`. Default is 1.


### `subscription.max_outstanding_messages` [_subscription_max_outstanding_messages]

The maximum number of unprocessed messages (unacknowledged but not yet expired). If the value is negative, then there will be no limit on the number of unprocessed messages. Due to the presence of internal queue, the input gets blocked until `queue.mem.flush.min_events` or `queue.mem.flush.timeout` is reached. To prevent this blockage, this option must be at least `queue.mem.flush.min_events`. Default is 1600.


### `credentials_file` [_credentials_file]

Path to a JSON file containing the credentials and key used to subscribe. As an alternative you can use the `credentials_json` config option or rely on [Google Application Default Credentials](https://cloud.google.com/docs/authentication/production) (ADC).


### `credentials_json` [_credentials_json]

JSON blob containing the credentials and key used to subscribe. This can be as an alternative to `credentials_file` if you want to embed the credential data within your config file or put the information into a keystore. You may also use [Google Application Default Credentials](https://cloud.google.com/docs/authentication/production) (ADC).


## Common options [filebeat-input-gcp-pubsub-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_10]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_10]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: gcp-pubsub
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-gcp-pubsub-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: gcp-pubsub
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-gcp-pubsub]

If this option is set to true, the custom [fields](#filebeat-input-gcp-pubsub-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_10]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_10]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_10]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_10]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_10]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


## Metrics [_metrics_9]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `acked_message_total` | Number of successfully ACKed messages. |
| `failed_acked_message_total` | Number of failed ACKed messages. |
| `nacked_message_total` | Number of NACKed messages. |
| `bytes_processed_total` | Number of bytes processed. |
| `processing_time` | Histogram of the elapsed time for processing an event in nanoseconds. |


