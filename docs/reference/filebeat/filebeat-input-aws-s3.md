---
navigation_title: "AWS S3"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-aws-s3.html
---

# AWS S3 input [filebeat-input-aws-s3]


Use the `aws-s3` input to retrieve logs from S3 objects that are pointed to by S3 notification events read from an SQS queue or directly polling list of S3 objects in an S3 bucket.  The use of SQS notification is preferred: polling lists of S3 objects is expensive in terms of performance and costs and should be preferably used only when no SQS notification can be attached to the S3 buckets. This input can, for example, be used to receive S3 access logs to monitor detailed records for the requests that are made to a bucket. This input also supports S3 notification from SNS to SQS.

SQS notification method is enabled setting `queue_url` configuration value.  S3 bucket list polling method is enabled setting `bucket_arn` configuration value. Both values cannot be set at the same time, at least one of the values must be set.

When using the SQS notification method, this input depends on S3 notifications delivered to an SQS queue for `s3:ObjectCreated:*` events. You must create an SQS queue and configure S3 to publish events to the queue.

The S3 input manages SQS message visibility to prevent messages from being reprocessed while the S3 object is still being processed. If the processing takes longer than half of the visibility timeout, the timeout is reset to ensure the message doesn’t return to the queue before processing is complete.

If an error occurs during the processing of the S3 object, the processing will be stopped, and the SQS message will be returned to the queue for reprocessing.


## Configuration Examples [_configuration_examples]


### SQS with JSON files [_sqs_with_json_files]

This example reads s3:ObjectCreated notifications from SQS, and assumes that all the S3 objects have a `Content-Type` of `application/json`. It splits the `Records` array in the JSON into separate events.

```yaml
filebeat.inputs:
- type: aws-s3
  queue_url: https://sqs.ap-southeast-1.amazonaws.com/1234/test-s3-queue
  expand_event_list_from_field: Records
```


### S3 bucket listing [_s3_bucket_listing]

When using the direct polling list of S3 objects in an S3 buckets, a number of workers that will process the S3 objects listed must be set through the `number_of_workers` config. Listing of the S3 bucket will be polled according the time interval defined by `bucket_list_interval` config. The default value is 120 sec.

```yaml
filebeat.inputs:
- type: aws-s3
  bucket_arn: arn:aws:s3:::test-s3-bucket
  number_of_workers: 5
  bucket_list_interval: 300s
  credential_profile_name: elastic-beats
  expand_event_list_from_field: Records
```


### S3-compatible services [_s3_compatible_services]

The `aws-s3` input can also poll third party S3-compatible services such as the Minio. Using non-AWS S3 compatible buckets requires the use of `access_key_id` and `secret_access_key` for authentication.  To specify the S3 bucket name, use the `non_aws_bucket_name` config and the `endpoint` must be set to replace the default API endpoint.  `endpoint` should be a full URI in the form of `https(s)://<s3 endpoint>` in the case of `non_aws_bucket_name`, that will be used as the API endpoint of the service.  No `endpoint` is needed if using the native AWS S3 service hosted at `amazonaws.com`.  Please see [Configuration parameters](#aws-credentials-config) for alternate AWS domains that require a different endpoint.

```yaml
filebeat.inputs:
- type: aws-s3
  non_aws_bucket_name: test-s3-bucket
  number_of_workers: 5
  bucket_list_interval: 300s
  access_key_id: xxxxxxx
  secret_access_key: xxxxxxx
  endpoint: https://s3.example.com:9000
  expand_event_list_from_field: Records
```


## Document ID Generation [_document_id_generation]

This aws-s3 input feature prevents the duplication of events in Elasticsearch by generating a custom document `_id` for each event, rather than relying on Elasticsearch to automatically generate one. Each document in an Elasticsearch index must have a unique `_id`, and Filebeat uses this property to avoid ingesting duplicate events.

The custom `_id` is based on several pieces of information from the S3 object: the Last-Modified timestamp, the bucket ARN, the object key, and the byte offset of the data in the event.

Duplicate prevention is particularly useful in scenarios where Filebeat needs to retry an operation. Filebeat guarantees at-least-once delivery, meaning it will retry any failed or incomplete operations. These retries may be triggered by issues with the host, `Filebeat`, network connectivity, or services such as Elasticsearch, SQS, or S3.


### Limitations of `_id`-Based Deduplication [_limitations_of_id_based_deduplication]

There are some limitations to consider when using `_id`-based deduplication in Elasticsearch:

* Deduplication works only within a single index. The same `_id` can exist in different indices, which is important if you’re using data streams or index aliases. When the backing index rolls over, a duplicate may be ingested.
* Indexing operations in Elasticsearch may take longer when an `_id` is specified. Elasticsearch needs to check if the ID already exists before writing, which can increase the time required for indexing.


### Disabling Duplicate Prevention [_disabling_duplicate_prevention]

If you want to disable the `_id`-based deduplication, you can remove the document `_id` using the [`drop_fields`](/reference/filebeat/drop-fields.md) processor in Filebeat.

```yaml
filebeat.inputs:
  - type: aws-s3
    queue_url: https://queue.amazonaws.com/80398EXAMPLE/MyQueue
    processors:
      - drop_fields:
          fields:
            - '@metadata._id'
          ignore_missing: true
```

Alternatively, you can remove the `_id` field using an Elasticsearch Ingest Node pipeline.

```json
{
  "processors": [
    {
      "remove": {
        "if": "ctx.input?.type == \"aws-s3\"",
        "field": "_id",
        "ignore_missing": true
      }
    }
  ]
}
```


## Handling Compressed Objects [_handling_compressed_objects]

S3 objects that use the gzip format ([RFC 1952](https://rfc-editor.org/rfc/rfc1952.html)) with the DEFLATE compression algorithm are automatically decompressed during processing. This is achieved by checking for the gzip file magic header.


## Configuration [_configuration]

The `aws-s3` input supports the following configuration options plus the [Common options](#filebeat-input-aws-s3-common-options) described later.

::::{note}
For time durations, valid time units are - "ns", "us" (or "µs"), "ms", "s", "m", "h". For example, "2h"
::::



### `api_timeout` [_api_timeout_2]

The maximum duration of the AWS API call. If it exceeds the timeout, the AWS API call will be interrupted. The default AWS API timeout is `120s`.

The API timeout must be longer than the `sqs.wait_time` value.


### `buffer_size` [input-aws-s3-buffer_size]

The size  of the buffer in bytes that each harvester uses when fetching a file. This only applies to non-JSON logs. The default is `16 KiB`.


### `content_type` [input-aws-s3-content_type]

A standard MIME type describing the format of the object data.  This can be set to override the MIME type given to the object when it was uploaded. For example: `application/json`.


### `encoding` [input-aws-s3-encoding]

The file encoding to use for reading data that contains international characters. This only applies to non-JSON logs. See [`encoding`](/reference/filebeat/filebeat-input-log.md#_encoding_3).


### `decoding` [input-aws-s3-decoding]

The file decoding option is used to specify a codec that will be used to decode the file contents. This can apply to any file stream data. An example config is shown below:

Currently supported codecs are given below:-

1. [csv](#attrib-decoding-csv): This codec decodes RFC 4180 CSV data streams.
2. [parquet](#attrib-decoding-parquet): This codec decodes Apache Parquet data streams.


#### `csv` [attrib-decoding-csv]

The CSV codec is used to decode RFC 4180 CSV data streams. Enabling the codec without other options will use the default codec options.

```yaml
  decoding.codec.csv.enabled: true
```

The `csv` codec supports five sub attributes to control aspects of CSV decoding. The `comma` attribute specifies the field separator character used by the CSV format. If it is not specified, the comma character *`,`* is used. The `comment` attribute specifies the character that should be interpreted as a comment mark.  If it is specified, lines starting with the character will be ignored. Both `comma` and `comment` must be single characters. The `lazy_quotes` attribute controls how quoting in fields is handled. If `lazy_quotes` is true, a quote may appear in an unquoted field and a non-doubled quote may appear in a quoted field.  The `trim_leading_space` attribute specifies that leading white space should be ignored, even if the `comma` character is white space. For complete details of the preceding configuration attribute behaviors, see the CSV decoder [documentation](https://pkg.go.dev/encoding/csv#Reader) The `fields_names` attribute can be used to specify the column names for the data. If it is absent, the field names are obtained from the first non-comment line of data. The number of fields must match the number of field names.

An example config is shown below:

```yaml
  decoding.codec.csv.enabled: true
  decoding.codec.csv.comma: "\t"
  decoding.codec.csv.comment: "#"
```


#### `parquet` [attrib-decoding-parquet]

The `parquet` codec is used to decode the [Apache Parquet](https://en.wikipedia.org/wiki/Apache_Parquet) data storage format. Enabling the codec without other options will use the default codec options.

```yaml
  decoding.codec.parquet.enabled: true
```

The Parquet codec supports two attributes, batch_size and process_parallel, to improve decoding performance:

* `batch_size`: This attribute specifies the number of records to read from the Parquet stream at a time. By default, batch_size is set to 1. Increasing the batch size can boost processing speed by reading more records in each operation.
* `process_parallel`: When set to true, this attribute allows Filebeat to read multiple columns from the Parquet stream in parallel, using as many readers as there are columns. Enabling parallel processing can significantly increase throughput, but it will also result in higher memory usage. By default, process_parallel is set to false.

By adjusting both batch_size and process_parallel, you can fine-tune the trade-off between processing speed and memory consumption.

An example config is shown below:

```yaml
  decoding.codec.parquet.enabled: true
  decoding.codec.parquet.process_parallel: true
  decoding.codec.parquet.batch_size: 1000
```


### `expand_event_list_from_field` [_expand_event_list_from_field]

If the fileset using this input expects to receive multiple messages bundled under a specific field or an array of objects then the config option `expand_event_list_from_field` value can be assigned the name of the field or `.[]`. This setting will be able to split the messages under the group value into separate events. For example, CloudTrail logs are in JSON format and events are found under the JSON object "Records".

::::{note}
When using `expand_event_list_from_field`, `content_type` config parameter has to be set to `application/json`.
::::


```json
{
    "Records": [
        {
            "eventVersion": "1.07",
            "eventTime": "2019-11-14T00:51:00Z",
            "awsRegion": "us-east-1",
            "eventID": "EXAMPLE8-9621-4d00-b913-beca2EXAMPLE",
        },
        {
            "eventVersion": "1.07",
            "eventTime": "2019-11-14T00:52:00Z",
            "awsRegion": "us-east-1",
            "eventID": "EXAMPLEc-28be-486c-8928-49ce6EXAMPLE",
        }
    ]
}
```

Or when `expand_event_list_from_field` is set to `.[]`, an array of objects will be split into separate events.

```json
[
   {
      "id":"1234",
      "message":"success"
   },
   {
      "id":"5678",
      "message":"failure"
   }
]
```

Note: When `expand_event_list_from_field` parameter is given in the config, aws-s3 input will assume the logs are in JSON format and decode them as JSON. Content type will not be checked. If a file has "application/json" content-type, `expand_event_list_from_field` becomes required to read the JSON file.


### `file_selectors` [_file_selectors]

If the SQS queue will have events that correspond to files that Filebeat shouldn’t process `file_selectors` can be used to limit the files that are downloaded.  This is a list of selectors which are made up of `regex` and `expand_event_list_from_field` options.  The `regex` should match the S3 object key in the SQS message, and the optional `expand_event_list_from_field` is the same as the global setting.  If `file_selectors` is given, then any global `expand_event_list_from_field` value is ignored in favor of the ones specified in the `file_selectors`.  Regex syntax is the same as the Go language.  Files that don’t match one of the regexes won’t be processed.  [`content_type`](#input-aws-s3-content_type), [`parsers`](#input-aws-s3-parsers), [`include_s3_metadata`](#input-aws-s3-include_s3_metadata),[`max_bytes`](#input-aws-s3-max_bytes), [`buffer_size`](#input-aws-s3-buffer_size), and [`encoding`](#input-aws-s3-encoding) may also be set for each file selector.

```yaml
file_selectors:
  - regex: '/CloudTrail/'
    expand_event_list_from_field: 'Records'
  - regex: '/CloudTrail-Digest/'
  - regex: '/CloudTrail-Insight/'
    expand_event_list_from_field: 'Records'
```


### `fips_enabled` [_fips_enabled]

Moved to [AWS credentials options](#aws-credentials-config).


### `include_s3_metadata` [input-aws-s3-include_s3_metadata]

This input can include S3 object metadata in the generated events for use in follow-on processing. You must specify the list of keys to include. By default, none are included. If the key exists in the S3 response, then it will be included in the event as `aws.s3.metadata.<key>` where the key name as been normalized to all lowercase.

```
include_s3_metadata:
  - last-modified
  - x-amz-version-id
```


### `max_bytes` [input-aws-s3-max_bytes]

The maximum number of bytes that a single log message can have. All bytes after `max_bytes` are discarded and not sent. This setting is especially useful for multiline log messages, which can get large. This only applies to non-JSON logs.  The default is `10 MiB`.


### `parsers` [input-aws-s3-parsers]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This option expects a list of parsers that non-JSON logs go through.

Available parsers:

* `multiline`

In this example, Filebeat is reading multiline messages that consist of XML that start with the `<Event>` tag.

```yaml
filebeat.inputs:
- type: aws-s3
  ...
  parsers:
    - multiline:
        pattern: "^<Event"
        negate:  true
        match:   after
```

See the available parser settings in detail below.


#### `multiline` [_multiline]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.


### `queue_url` [_queue_url]

URL of the AWS SQS queue that messages will be received from. (Required when `bucket_arn`, `access_point_arn`, and `non_aws_bucket_name` are not set).


### `region` [_region]

The name of the AWS region of the end point. If this option is given it takes precedence over the region name obtained from the `queue_url` value.


### `visibility_timeout` [_visibility_timeout]

The duration that the received SQS messages are hidden from retrieve requests after being retrieved by a `ReceiveMessage` request. The default visibility timeout is `300s`. The maximum is `12h`. Filebeat will automatically reset the visibility timeout of a message after 1/2 of the duration passes to prevent a message that is still being processed from returning to the queue.


### `sqs.max_receive_count` [_sqs_max_receive_count]

The maximum number of times a SQS message should be received (retried) before deleting it. This feature prevents poison-pill messages (messages that can be received but can’t be processed) from consuming resources. The number of times a message has been received is tracked using the `ApproximateReceiveCount` SQS attribute. The default value is 5.

If you have configured a dead letter queue, then you can set this value to `-1` to disable deletion on failure.


### `sqs.notification_parsing_script.source` [_sqs_notification_parsing_script_source]

Inline Javascript source code.

```yaml
sqs.notification_parsing_script.source: >
  function parse(notification) {
      var evts = [];
      var evt = new S3EventV2();
      evt.SetS3BucketName(notification.bucket);
      evt.SetS3ObjectKey(notification.path);
      evts.push(evt);
      return evts;
  }
```


### `sqs.notification_parsing_script.file` [_sqs_notification_parsing_script_file]

Path to a script file to load. Relative paths are interpreted as relative to the `path.config` directory. Globs are expanded.

This loads `filter.js` from disk.

```yaml
sqs.notification_parsing_script.file: ${path.config}/filter.js
```


### `sqs.notification_parsing_script.files` [_sqs_notification_parsing_script_files]

List of script files to load. The scripts are concatenated together. Relative paths are interpreted as relative to the `path.config` directory. And globs are expanded.


### `sqs.notification_parsing_script.params` [_sqs_notification_parsing_script_params]

A dictionary of parameters that are passed to the `register` of the script.

Parameters can be passed to the script by adding `params` to the config. This allows for a script to be made reusable. When using `params` the code must define a `register(params)` function to receive the parameters.

```yaml
sqs.notification_parsing_script:
  params:
    provider: aws:s3
  source: >
    var params = {provider: ""};
    function register(scriptParams) {
      params = scriptParams;
    }
    function parse(notification) {
      var evts = [];
      var evt = new S3EventV2();
      evt.SetS3BucketName(notification.bucket);
      evt.SetS3ObjectKey(notification.path);
      evt.SetProvider(params.provider);
      evts.push(evt);
      return evts;
    }
```


### `sqs.notification_parsing_script.timeout` [_sqs_notification_parsing_script_timeout]

This sets an execution timeout for the `process` function. When the `process` function takes longer than the `timeout` period the function is interrupted. You can set this option to prevent a script from running for too long (like preventing an infinite `while` loop). By default, there is no timeout.


### `sqs.notification_parsing_script.max_cached_sessions` [_sqs_notification_parsing_script_max_cached_sessions]

This sets the maximum number of JavaScript VM sessions that will be cached to avoid reallocation.


### `sqs.wait_time` [_sqs_wait_time]

The maximum duration that an SQS `ReceiveMessage` call should wait for a message to arrive in the queue before returning. The default value is `20s`. The maximum value is `20s`.


### `sqs.shutdown_grace_time` [_shutdown_grace_time]

The duration that an SQS message processor will wait for a messages to arrive in the queue and be processed before allowing the input to terminate when a cancelation has been received. The default value is `20s`. It must not be negative.


### `bucket_arn` [_bucket_arn]

ARN of the AWS S3 bucket that will be polled for list operation. (Required when `queue_url`, `access_point_arn, and `non_aws_bucket_name` are not set).


### `access_point_arn` [_access_point_arn]

ARN of the AWS S3 Access Point that will be polled for list operation. (Required when `queue_url`, `bucket_arn`, and `non_aws_bucket_name` are not set).


### `non_aws_bucket_name` [_non_aws_bucket_name]

Name of the S3 bucket that will be polled for list operation. Required for third-party S3 compatible services. (Required when `queue_url` and `bucket_arn` are not set).


### `bucket_list_interval` [_bucket_list_interval]

Time interval for polling listing of the S3 bucket: default to `120s`.


### `bucket_list_prefix` [_bucket_list_prefix]

Prefix to apply for the list request to the S3 bucket. Default empty.


### `number_of_workers` [_number_of_workers_2]

Number of workers that will process the S3 or SQS objects listed. Required when `bucket_arn` or `access_point_arn` is set, otherwise (in the SQS case) defaults to 5.


### `provider` [_provider]

Name of the third-party S3 bucket provider like backblaze or GCP. The following endpoints/providers will be detected automatically:

|     |     |
| --- | --- |
| Domain | Provider |
| amazonaws.com, amazonaws.com.cn, c2s.sgov.gov, c2s.ic.gov | aws |
| backblazeb2.com | backblaze |
| wasabisys.com | wasabi |
| digitaloceanspaces.com | digitalocean |
| dream.io | dreamhost |
| scw.cloud | scaleway |
| googleapis.com | gcp |
| cloud.it | arubacloud |
| linodeobjects.com | linode |
| vultrobjects.com | vultr |
| appdomain.cloud | ibm |
| aliyuncs.com | alibaba |
| oraclecloud.com | oracle |
| exo.io | exoscale |
| upcloudobjects.com | upcloud |
| ilandcloud.com | iland |
| zadarazios.com | zadara |


### `path_style` [_path_style]

Enabling this option sets the bucket name as a path in the API call instead of a subdomain. When enabled `https://<bucket-name>.s3.<region>.<provider>.com` becomes `https://s3.<region>.<provider>.com/<bucket-name>`.  This is only supported with third-party S3 providers.  AWS does not support path style.


### `aws credentials` [_aws_credentials_2]

To make AWS API calls, `aws-s3` input requires AWS credentials. Please see [AWS credentials options](#aws-credentials-config) for more details.


### `backup_to_bucket_arn` [_backup_to_bucket_arn]

The ARN of the S3 bucket where processed files are copied. The copy is created after the S3 object is fully processed. When using the `non_aws_bucket_name`, please use `non_aws_backup_to_bucket_name` accordingly.

Naming of the backed up files can be controlled with `backup_to_bucket_prefix`.


### `backup_to_bucket_prefix` [_backup_to_bucket_prefix]

This prefix will be prepended to the object key when backing it up to another (or the same) bucket.


### `non_aws_backup_to_bucket_name` [_non_aws_backup_to_bucket_name]

The name of the non-AWS bucket where processed files are copied. Use this parameter when not using AWS buckets. The copy is created after the S3 object is fully processed.  When using the `bucket_arn`, please use `backup_to_bucket_arn` accordingly.

Naming of the backed up files can be controlled with `backup_to_bucket_prefix`.


### `delete_after_backup` [_delete_after_backup]

Controls whether fully processed files will be deleted from the bucket.

This option can only be used together with the backup functionality.

## Common options [filebeat-input-aws-s3-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_2]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_2]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: aws-s3
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-aws-s3-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: aws-s3
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-aws-s3]

If this option is set to true, the custom [fields](#filebeat-input-aws-s3-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_2]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_2]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_2]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_2]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_2]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


### `ignore_older` [_ignore_older]

The parameter specifies the time duration (ex:- 30m, 2h or 48h) during which bucket entries are accepted for processing. By default, this feature is disabled, allowing any entry in the bucket to be processed. It is recommended to set a suitable duration to prevent older bucket entries from being tracked, which helps to reduce the memory usage.

When defined, bucket entries are processed only if their last modified timestamp falls within the specified time duration, relative to the current time. However, when the start_timestamp is set, the initial processing will include all bucket entries up to that timestamp.

::::{note}
Bucket entries that are older than the defined duration and have failed processing will not be re-processed. It is recommended to configure a sufficiently long duration based on your use case and current settings to avoid conflicts with the bucket_list_interval property. Additionally, this ensures that subsequent runs can include and re-process objects that failed due to unavoidable errors.
::::



### `start_timestamp` [_start_timestamp]

Accepts a timestamp in the YYYY-MM-DDTHH:MM:SSZ format, which defines the point from which bucket entries are accepted for processing. By default, this is disabled, allowing all entries in the bucket to be processed.

This parameter is useful when configuring input for the first time, especially if you want to ingest logs starting from a specific time. The timestamp can also be set to a future time, offering greater flexibility. You can combine this property with ignore_older duration to improve memory usage by reducing tracked bucket entries.

::::{note}
It is recommended to update this value when updating or restarting filebeat
::::



## AWS Permissions [_aws_permissions_2]

Specific AWS permissions are required for IAM user to access SQS and S3 when using the SQS notifications method:

```
s3:GetObject
sqs:ReceiveMessage
sqs:ChangeMessageVisibility
sqs:DeleteMessage
```

Reduced specific S3 AWS permissions are required for IAM user to access S3 when using the polling list of S3 bucket objects:

```
s3:GetObject
s3:ListBucket
s3:GetBucketLocation
```

In case `backup_to_bucket_arn` or `non_aws_backup_to_bucket_name` are set the following permission is required as well:

```
s3:PutObject
```

In case `delete_after_backup` is set the following permission is required as well:

```
s3:DeleteObject
```

In case optional SQS metric `sqs_messages_waiting_gauge` is desired, the following permission is required:

```
sqs:GetQueueAttributes
```


## S3 and SQS setup [_s3_and_sqs_setup]

To configure SQS notifications for an existing S3 bucket, you can follow [create-sqs-queue-for-notification](https://docs.aws.amazon.com/AmazonS3/latest/dev/ways-to-add-notification-config-to-bucket.html#step1-create-sqs-queue-for-notification) guide.

Alternatively, you can follow steps given which use a CloudFormation template to create a S3 bucket connected to a SQS with object creation notifications already enabled.

1. First copy the CloudFormation template given below to a desired location. For example, to file `awsCloudFormation.yaml`

    ::::{dropdown} CloudFormation template
    ```yaml
    AWSTemplateFormatVersion: '2010-09-09'
    Description: |
      Create a S3 bucket connected to a SQS for filebeat validations
    Resources:
      S3BucketWithSQS:
        Type: AWS::S3::Bucket
        Properties:
          BucketName: !Sub ${AWS::StackName}-s3bucket
          BucketEncryption:
            ServerSideEncryptionConfiguration:
              - ServerSideEncryptionByDefault:
                  SSEAlgorithm: aws:kms
                  KMSMasterKeyID: alias/aws/s3
          PublicAccessBlockConfiguration:
            IgnorePublicAcls: true
            RestrictPublicBuckets: true
          NotificationConfiguration:
            QueueConfigurations:
              - Event: s3:ObjectCreated:*
                Queue: !GetAtt SQSWithS3BucketConnected.Arn
        DependsOn:
          - S3BucketWithSQSToSQSWithS3BucketConnectedPermission
      S3BucketWithSQSBucketPolicy:
        Type: AWS::S3::BucketPolicy
        Properties:
          Bucket: !Ref S3BucketWithSQS
          PolicyDocument:
            Id: RequireEncryptionInTransit
            Version: '2012-10-17'
            Statement:
              - Principal: '*'
                Action: '*'
                Effect: Deny
                Resource:
                  - !GetAtt S3BucketWithSQS.Arn
                  - !Sub ${S3BucketWithSQS.Arn}/*
                Condition:
                  Bool:
                    aws:SecureTransport: 'false'
      SQSWithS3BucketConnected:
        Type: AWS::SQS::Queue
        Properties:
          MessageRetentionPeriod: 345600
      S3BucketWithSQSToSQSWithS3BucketConnectedPermission:
        Type: AWS::SQS::QueuePolicy
        Properties:
          PolicyDocument:
            Version: '2012-10-17'
            Statement:
              - Effect: Allow
                Principal:
                  Service: s3.amazonaws.com
                Action: sqs:SendMessage
                Resource: !GetAtt SQSWithS3BucketConnected.Arn
                Condition:
                  ArnEquals:
                    aws:SourceArn: !Sub arn:${AWS::Partition}:s3:::${AWS::StackName}-s3bucket
          Queues:
            - !Ref SQSWithS3BucketConnected
    Outputs:
      S3BucketArn:
        Description: The ARN of the S3 bucket to insert logs
        Value: !GetAtt S3BucketWithSQS.Arn
      SQSUrl:
        Description: The SQS URL to use for filebeat
        Value: !GetAtt SQSWithS3BucketConnected.QueueUrl
    ```

    ::::

2. Next, create a CloudFormation stack sourcing the copied.

    ```sh
    aws cloudformation create-stack --stack-name <STACK_NAME> --template-body file://awsCloudFormation.yaml
    ```

3. Then, obtain the S3 bucket ARN and SQS queue url using stack’s output

    For this, you can describe the stack created above. The S3 ARN is set to `S3BucketArn` output and SQS url is set to `SQSUrl` output.  The output will be populated once the `StackStatus` is set to `CREATE_COMPLETE`.

    ```sh
    aws cloudformation describe-stacks --stack-name <STACK_NAME>
    ```

4. Finally, you can configure filebeat to use SQS notifications

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: <URL_FROM_STACK>
      expand_event_list_from_field: Records
      credential_profile_name: elastic-beats
    ```

    With this configuration, Filebeat avoids polling and uses SQS notifications to extract logs from the S3 bucket.



## S3 → SNS → SQS setup [_s3_sns_sqs_setup]

If you would like to use the bucket notification in multiple different consumers (others than filebeat), you should use an SNS topic for the bucket notification.  Please see [create-SNS-topic-for-notification](https://docs.aws.amazon.com/AmazonS3/latest/userguide/ways-to-add-notification-config-to-bucket.html#step1-create-sns-topic-for-notification) for more details. SQS queue will be configured as a [subscriber to the SNS topic](https://docs.aws.amazon.com/sns/latest/dg/sns-sqs-as-subscriber.html).


## S3 → EventBridge → SQS setup [_s3_eventbridge_sqs_setup]

Amazon S3 can alternatively [send events to EventBridge](https://docs.aws.amazon.com/AmazonS3/latest/userguide/EventBridge.html), which can then be used to route these events to SQS. While the S3 input will filter for *Object Created* events, it is more efficient to configure EventBridge to only forward the *Object Created* events.


## Parallel Processing [_parallel_processing]

When using the SQS notifications method, multiple Filebeat instances can read from the same SQS queues at the same time.  To horizontally scale processing when there are large amounts of log data flowing into an S3 bucket, you can run multiple Filebeat instances that read from the same SQS queues at the same time. No additional configuration is required.

Using SQS ensures that each message in the queue is processed only once even when multiple Filebeat instances are running in parallel. To prevent Filebeat from receiving and processing the message more than once, set the visibility timeout.

The visibility timeout begins when SQS returns a message to Filebeat. During this time, Filebeat processes and deletes the message. However, if Filebeat fails before deleting the message and your system doesn’t call the DeleteMessage action for that message before the visibility timeout expires, the message becomes visible to other Filebeat instances, and the message is received again. By default, the visibility timeout is set to 5 minutes for aws-s3 input in Filebeat. 5 minutes is sufficient time for Filebeat to read SQS messages and process related s3 log files.

When using the polling list of S3 bucket objects method be aware that if running multiple Filebeat instances, they can list the same S3 bucket at the same time. Since the state of the ingested S3 objects is persisted (upon processing a single list operation) in the `path.data` configuration and multiple Filebeat cannot share the same `path.data` this will produce repeated ingestion of the S3 object.  Therefore, when using the polling list of S3 bucket objects method, scaling should be vertical, with a single bigger Filebeat instance and higher `number_of_workers` config value.


## SQS Custom Notification Parsing Script [_sqs_custom_notification_parsing_script]

Under some circumstances, you might want to listen to events that are not following the standard SQS notifications format. To be able to parse them, it is possible to define a custom script that will take care of processing them and generating the required list of S3 Events used to download the files.

The `sqs.notification_parsing_script` executes JavaScript code to process an event.  It uses a pure Go implementation of ECMAScript 5.1 and has no external dependencies.

It can be configured by embedding JavaScript in your configuration file or by pointing the processor at external file(s). Only one of the options `sqs.notification_parsing_script.source`, `sqs.notification_parsing_script.file`, and `sqs.notification_parsing_script.files` can be set at the same time.

The script requires a `parse(notification)` function that receives the notification as a raw string and returns a list of `S3EventV2` objects. This raw string can then be processed as needed, e.g.: `JSON.parse(n)` or the provided helper for XML `new XMLDecoder(n)`.

If the script defines a `test()` function it will be invoked when it is loaded. Any exceptions thrown will cause the processor to fail to load. This can be used to make assertions about the behavior of the script.

```javascript
function parse(n) {
  var m = JSON.parse(n);
  var evts = [];
  var files = m.files;
  var bucket = m.bucket;

  if (!Array.isArray(files) || (files.length == 0) || bucket == null || bucket == "") {
    return evts;
  }

  files.forEach(function(f){
    var evt = new S3EventV2();
    evt.SetS3BucketName(bucket);
    evt.SetS3ObjectKey(f.path);
    evts.push(evt);
  });

  return evts;
}

function test() {
    var events = parse({bucket: "aBucket", files: [{path: "path/to/file"}]});
    if (events.length !== 1) {
      throw "expecting one event";
    }
    if (events[0].S3.Bucket.Name === "aBucket") {
        throw "expected bucket === aBucket";
    }
    if (events[0].S3.Object.Key === "path/to/file") {
        throw "expected bucket === path/to/file";
    }
}
```


### S3EventV2 API [_s3eventv2_api]

The `S3EventV2` object returned by the `parse` method.

| Method | Description |
| --- | --- |
| `new S3EventV2()` | Returns a new `S3EventV2` object.<br>**Example**: `var evt = new S3EventV2();` |
| `SetAWSRegion(string)` | Sets the AWS region.<br>**Example**: `evt.SetAWSRegion("us-east-1");` |
| `SetProvider(string)` | Sets the provider.<br>**Example**: `evt.SetProvider("provider");` |
| `SetEventName(string)` | Sets the event name.<br>**Example**: `evt.SetEventName("event-type");` |
| `SetEventSource(string)` | Sets the event surce.<br>**Example**: `evt.SetEventSource("aws:s3");` |
| `SetS3BucketName(string)` | Sets the bucket name.<br>**Example**: `evt.SetS3BucketName("bucket-name");` |
| `SetS3BucketARN(string)` | Sets the bucket ARN.<br>**Example**: `evt.SetS3BucketARN("bucket-ARN");` |
| `SetS3ObjectKey(string)` | Sets the object key.<br>**Example**: `evt.SetS3ObjectKey("path/to/object");` |

To be able to retrieve an S3 object successfully, at least `S3.Object.Key` and `S3.Bucket.Name` properties must be set (using the provided setters). The other properties will be used as metadata in the resulting event when available.


### XMLDecoder API [_xmldecoder_api]

To help with XML decoding, an `XMLDecoder` class is provided.

Example XML input:

```xml
<catalog>
  <book seq="1">
    <author>William H. Gaddis</author>
    <title>The Recognitions</title>
    <review>One of the great seminal American novels of the 20th century.</review>
  </book>
</catalog>
```

Will produce the following output:

```json
{
  "catalog": {
    "book": {
      "author": "William H. Gaddis",
      "review": "One of the great seminal American novels of the 20th century.",
      "seq": "1",
      "title": "The Recognitions"
    }
  }
}
```

| Method | Description |
| --- | --- |
| `new XMLDecoder(string)` | Returns a new `XMLDecoder` object to decode the provided `string`.<br>**Example**: `var dec = new XMLDecoder(n);` |
| `PrependHyphenToAttr()` | Causes the Decoder to prepend a hyphen (`-`) to all XML attribute names.<br>**Example**: `dec.PrependHyphenToAttr();` |
| `LowercaseKeys()` | Causes the Decoder to transform all key names to lowercase.<br>**Example**: `dec.LowercaseKeys();` |
| `Decode()` | Reads the XML string and return a map containing the data.<br>**Example**: `var m = dec.Decode();` |


## AWS Credentials Configuration [aws-credentials-config]

To configure AWS credentials, either put the credentials into the Filebeat configuration, or use a shared credentials file, as shown in the following examples.


### Configuration parameters [_configuration_parameters]

* **access_key_id**: first part of access key.
* **secret_access_key**: second part of access key.
* **session_token**: required when using temporary security credentials.
* **credential_profile_name**: profile name in shared credentials file.
* **shared_credential_file**: directory of the shared credentials file.
* **role_arn**: AWS IAM Role to assume.
* **external_id**: external ID to use when assuming a role in another account, see [the AWS documentation for use of external IDs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html).
* **proxy_url**: URL of the proxy to use to connect to AWS web services. The syntax is `http(s)://<IP/Hostname>:<port>`
* **fips_enabled**: Enabling this option instructs Filebeat to use the FIPS endpoint of a service. All services used by Filebeat are FIPS compatible except for `tagging` but only certain regions are FIPS compatible. See [https://aws.amazon.com/compliance/fips/](https://aws.amazon.com/compliance/fips/) or the appropriate service page, [https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html](https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html), for a full list of FIPS endpoints and regions.
* **ssl**: This specifies SSL/TLS configuration. If the ssl section is missing, the host’s CAs are used for HTTPS connections. See [SSL](/reference/filebeat/configuration-ssl.md) for more information.
* **default_region**: Default region to query if no other region is set. Most AWS services offer a regional endpoint that can be used to make requests. Some services, such as IAM, do not support regions. If a region is not provided by any other way (environment variable, credential or instance profile), the value set here will be used.
* **assume_role.duration**: The duration of the requested assume role session. Defaults to 15m when not set. AWS allows a maximum session duration between 1h and 12h depending on your maximum session duration policies.
* **assume_role.expiry_window**: The expiry_window will allow refreshing the session prior to its expiration. This is beneficial to prevent expiring tokens from causing requests to fail with an ExpiredTokenException.


### Supported Formats [_supported_formats]

::::{note}
The examples in this section refer to Metricbeat, but the credential options for authentication with AWS are the same no matter which Beat is being used.
::::


* Use `access_key_id`, `secret_access_key`, and/or `session_token`

Users can either put the credentials into the Metricbeat module configuration or use environment variable `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and/or `AWS_SESSION_TOKEN` instead.

If running on Docker, these environment variables should be added as a part of the docker command. For example, with Metricbeat:

```bash
$ docker run -e AWS_ACCESS_KEY_ID=abcd -e AWS_SECRET_ACCESS_KEY=abcd -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  access_key_id: ${AWS_ACCESS_KEY_ID}
  secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  session_token: ${AWS_SESSION_TOKEN}
  metricsets:
    - ec2
```

Environment variables can also be added through a file. For example:

```bash
$ cat env.list
AWS_ACCESS_KEY_ID=abcd
AWS_SECRET_ACCESS_KEY=abcd

$ docker run --env-file env.list -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

* Use `credential_profile_name` and/or `shared_credential_file`

If `access_key_id`, `secret_access_key` and `role_arn` are all not given, then filebeat will check for `credential_profile_name`. If you use different credentials for different tools or applications, you can use profiles to configure multiple access keys in the same configuration file. If there is no `credential_profile_name` given, the default profile will be used.

`shared_credential_file` is optional to specify the directory of your shared credentials file. If it’s empty, the default directory will be used. In Windows, shared credentials file is at `C:\Users\<yourUserName>\.aws\credentials`. For Linux, macOS or Unix, the file is located at `~/.aws/credentials`. When running as a service, the home path depends on the user that manages the service, so the `shared_credential_file` parameter can be used to avoid ambiguity. Please see [Create Shared Credentials File](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/create-shared-credentials-file.md) for more details.

* Use `role_arn`

`role_arn` is used to specify which AWS IAM role to assume for generating temporary credentials. If `role_arn` is given, filebeat will check if access keys are given. If not, filebeat will check for credential profile name. If neither is given, default credential profile will be used. Please make sure credentials are given under either a credential profile or access keys.

If running on Docker, the credential file needs to be provided via a volume mount. For example, with Metricbeat:

```bash
docker run -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" --volume="/Users/foo/.aws/credentials:/usr/share/metricbeat/credentials:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  credential_profile_name: elastic-beats
  shared_credential_file: /usr/share/metricbeat/credentials
  metricsets:
    - ec2
```

* Use AWS credentials in Filebeat configuration

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      access_key_id: '<access_key_id>'
      secret_access_key: '<secret_access_key>'
      session_token: '<session_token>'
    ```

    or

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      access_key_id: '${AWS_ACCESS_KEY_ID:""}'
      secret_access_key: '${AWS_SECRET_ACCESS_KEY:""}'
      session_token: '${AWS_SESSION_TOKEN:""}'
    ```

* Use IAM role ARN

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      role_arn: arn:aws:iam::123456789012:role/test-mb
    ```

* Use shared AWS credentials file

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      credential_profile_name: test-fb
    ```



### AWS Credentials Types [_aws_credentials_types]

There are two different types of AWS credentials can be used: access keys and temporary security credentials.

* Access keys

`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are the two parts of access keys. They are long-term credentials for an IAM user or the AWS account root user. Please see [AWS Access Keys and Secret Access Keys](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys) for more details.

* IAM role ARN

An IAM role is an IAM identity that you can create in your account that has specific permissions that determine what the identity can and cannot do in AWS. A role does not have standard long-term credentials such as a password or access keys associated with it. Instead, when you assume a role, it provides you with temporary security credentials for your role session. IAM role Amazon Resource Name (ARN) can be used to specify which AWS IAM role to assume to generate temporary credentials. Please see [AssumeRole API documentation](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html) for more details.

Here are the steps to set up IAM role using AWS CLI for Metricbeat. Please replace `123456789012` with your own account ID.

Step 1. Create `example-policy.json` file to include all permissions:

```yaml
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "sqs:ReceiveMessage"
            ],
            "Resource": "*"
        },
        {
            "Sid": "VisualEditor1",
            "Effect": "Allow",
            "Action": "sqs:ChangeMessageVisibility",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor2",
            "Effect": "Allow",
            "Action": "sqs:DeleteMessage",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor3",
            "Effect": "Allow",
            "Action": [
                "sts:AssumeRole",
                "sqs:ListQueues",
                "tag:GetResources",
                "ec2:DescribeInstances",
                "cloudwatch:GetMetricData",
                "ec2:DescribeRegions",
                "iam:ListAccountAliases",
                "sts:GetCallerIdentity",
                "cloudwatch:ListMetrics"
            ],
            "Resource": "*"
        }
    ]
}
```

Step 2. Create IAM policy using the `aws iam create-policy` command:

```bash
$ aws iam create-policy --policy-name example-policy --policy-document file://example-policy.json
```

Step 3. Create the JSON file `example-role-trust-policy.json` that defines the trust relationship of the IAM role

```yaml
{
    "Version": "2012-10-17",
    "Statement": {
        "Effect": "Allow",
        "Principal": { "AWS": "arn:aws:iam::123456789012:root" },
        "Action": "sts:AssumeRole"
    }
}
```

Step 4. Create the IAM role and attach the policy:

```bash
$ aws iam create-role --role-name example-role --assume-role-policy-document file://example-role-trust-policy.json
$ aws iam attach-role-policy --role-name example-role --policy-arn "arn:aws:iam::123456789012:policy/example-policy"
```

After these steps are done, IAM role ARN can be used for authentication in Metricbeat `aws` module.

* Temporary security credentials

Temporary security credentials has a limited lifetime and consists of an access key ID, a secret access key, and a security token which typically returned from `GetSessionToken`. MFA-enabled IAM users would need to submit an MFA code while calling `GetSessionToken`. Please see [Temporary Security Credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) for more details. `sts get-session-token` AWS CLI can be used to generate temporary credentials. For example. with MFA-enabled:

```bash
aws> sts get-session-token --serial-number arn:aws:iam::1234:mfa/your-email@example.com --token-code 456789 --duration-seconds 129600
```

Because temporary security credentials are short term, after they expire, the user needs to generate new ones and modify the aws.yml config file with the new credentials. Unless [live reloading](/reference/metricbeat/_live_reloading.md) feature is enabled for Metricbeat, the user needs to manually restart Metricbeat after updating the config file in order to continue collecting Cloudwatch metrics. This will cause data loss if the config file is not updated with new credentials before the old ones expire. For Metricbeat, we recommend users to use access keys in config file to enable aws module making AWS api calls without have to generate new temporary credentials and update the config frequently.

IAM policy is an entity that defines permissions to an object within your AWS environment. Specific permissions needs to be added into the IAM user’s policy to authorize Metricbeat to collect AWS monitoring metrics. Please see documentation under each metricset for required permissions.


## Metrics [_metrics_2]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `sqs_messages_received_total` | Number of SQS messages received (not necessarily processed fully). |
| `sqs_visibility_timeout_extensions_total` | Number of SQS visibility timeout extensions. |
| `sqs_messages_inflight_gauge` | Number of SQS messages inflight (gauge). |
| `sqs_messages_returned_total` | Number of SQS messages returned to queue (happens on errors implicitly after visibility timeout passes). |
| `sqs_messages_deleted_total` | Number of SQS messages deleted. |
| `sqs_messages_waiting_gauge` | Number of SQS messages waiting in the SQS queue (gauge). The value is refreshed every minute via data from [https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_GetQueueAttributes.html<GetQueueAttributes&gt](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_GetQueueAttributes.md<GetQueueAttributes&gt);. A value of `-1` indicates the metric is uninitialized or could not be collected due to an error. |
| `sqs_worker_utilization` | Rate of SQS worker utilization over the previous 5 seconds. 0 indicates idle, 1 indicates all workers utilized. |
| `sqs_message_processing_time` | Histogram of the elapsed SQS processing times in nanoseconds (time of receipt to time of delete/return). |
| `sqs_lag_time` | Histogram of the difference between the SQS SentTimestamp attribute and the time when the SQS message was received expressed in nanoseconds. |
| `s3_objects_requested_total` | Number of S3 objects downloaded. |
| `s3_objects_listed_total` | Number of S3 objects returned by list operations. |
| `s3_objects_processed_total` | Number of S3 objects that matched file_selectors rules. |
| `s3_objects_acked_total` | Number of S3 objects processed that were fully ACKed. |
| `s3_bytes_processed_total` | Number of S3 bytes processed. |
| `s3_events_created_total` | Number of events created from processing S3 data. |
| `s3_objects_inflight_gauge` | Number of S3 objects inflight (gauge). |
| `s3_object_processing_time` | Histogram of the elapsed S3 object processing times in nanoseconds (start of download to completion of parsing). |


