---
navigation_title: "AWS CloudWatch"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-aws-cloudwatch.html
---

# AWS CloudWatch input [filebeat-input-aws-cloudwatch]


`aws-cloudwatch` input can be used to retrieve all logs from all log streams in a specific log group. `filterLogEvents` AWS API is used to list log events from the specified log group. Amazon CloudWatch Logs can be used to store log files from Amazon Elastic Compute Cloud(EC2), AWS CloudTrail, Route53, and other sources.

A log group is a group of log streams that share the same retention, monitoring, and access control settings. You can define log groups and specify which streams to put into each group. There is no limit on the number of log streams that can belong to one log group.

A log stream is a sequence of log events that share the same source. Each separate source of logs in CloudWatch Logs makes up a separate log stream.

```yaml
filebeat.inputs:
- type: aws-cloudwatch
  log_group_arn: arn:aws:logs:us-east-1:428152502467:log-group:test:*
  scan_frequency: 1m
  credential_profile_name: elastic-beats
  start_position: beginning
```

The `aws-cloudwatch` input supports the following configuration options plus the [Common options](#filebeat-input-aws-cloudwatch-common-options) described later.


### `log_group_arn` [_log_group_arn]

ARN of the log group to collect logs from. The ARN may refer to a log group in a linked source account.

Note: `log_group_arn` cannot be combined with `log_group_name`, `log_group_name_prefix` and `region_name` properties. If set, values extracted from `log_group_arn` takes precedence over them.

Note: If the log group is in a linked source account and filebeat is configured to use a monitoring account, you must use the `log_group_arn`. You can read more about AWS account linking and cross account observability from the [official documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-Unified-Cross-Account.html).


### `log_group_name` [_log_group_name]

Name of the log group to collect logs from.

Note: `region_name` is required when log_group_name is given.


### `log_group_name_prefix` [_log_group_name_prefix]

The prefix for a group of log group names. See `include_linked_accounts_for_prefix_mode` option for linked source accounts behavior.

Note: `region_name` is required when `log_group_name_prefix` is given. `log_group_name` and `log_group_name_prefix` cannot be given at the same time. The number of workers that will process the log groups under this prefix is set through the `number_of_workers` config.


### `include_linked_accounts_for_prefix_mode` [_include_linked_accounts_for_prefix_mode]

Configure whether to include linked source accounts that contains the prefix value defined through `log_group_name_prefix`. Accepts a boolean and this is by default disabled.

Note: Utilize `log_group_arn` if you desire to obtain logs from a known log group (including linked source accounts) You can read more about AWS account linking and cross account observability from the [official documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/CloudWatch-Unified-Cross-Account.html).


### `region_name` [_region_name]

Region that the specified log group or log group prefix belongs to.


### `number_of_workers` [_number_of_workers]

Number of workers that will process the log groups with the given `log_group_name_prefix`. Default value is 1.


### `log_streams` [_log_streams]

A list of strings of log streams names that Filebeat collect log events from.


### `log_stream_prefix` [_log_stream_prefix]

A string to filter the results to include only log events from log streams that have names starting with this prefix.


### `start_position` [_start_position]

`start_position` allows user to specify if this input should read log files from the `beginning` or from the `end`.

* `beginning`: reads from the beginning of the log group (default).
* `end`: read only new messages from current time minus `scan_frequency` going forward

For example, with `scan_frequency` equals to `30s` and current timestamp is `2020-06-24 12:00:00`:

* with `start_position = beginning`:

    * first iteration: startTime=0, endTime=2020-06-24 12:00:00
    * second iteration: startTime=2020-06-24 12:00:00, endTime=2020-06-24 12:00:30

* with `start_position = end`:

    * first iteration: startTime=2020-06-24 11:59:30, endTime=2020-06-24 12:00:00
    * second iteration: startTime=2020-06-24 12:00:00, endTime=2020-06-24 12:00:30



### `scan_frequency` [_scan_frequency]

This config parameter sets how often Filebeat checks for new log events from the specified log group. Default `scan_frequency` is 1 minute, which means Filebeat will sleep for 1 minute before querying for new logs again.


### `api_timeout` [_api_timeout]

The maximum duration of AWS API can take. If it exceeds the timeout, AWS API will be interrupted. The default AWS API timeout for a message is 120 seconds. The minimum is 0 seconds.


### `api_sleep` [_api_sleep]

This is used to sleep between AWS `FilterLogEvents` API calls inside the same collection period. `FilterLogEvents` API has a quota of 5 transactions per second (TPS)/account/Region. By default, `api_sleep` is 200 ms. This value should only be adjusted when there are multiple Filebeats or multiple Filebeat inputs collecting logs from the same region and AWS account.


### `latency` [_latency]

Some AWS services send logs to CloudWatch with a latency to process larger than `aws-cloudwatch` input `scan_frequency`. This case, please specify a `latency` parameter so collection start time and end time will be shifted by the given latency amount.


### `aws credentials` [_aws_credentials]

In order to make AWS API calls, `aws-cloudwatch` input requires AWS credentials. Please see [AWS credentials options](/reference/filebeat/filebeat-input-aws-s3.md#aws-credentials-config) for more details.


## AWS Permissions [_aws_permissions]

Specific AWS permissions are required for IAM user to access aws-cloudwatch:

```
cloudwatchlogs:DescribeLogGroups
logs:FilterLogEvents
```


## Metrics [_metrics]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input.

| Metric | Description |
| --- | --- |
| `log_events_received_total` | Number of CloudWatch log events received. |
| `log_groups_total` | Logs collected from number of CloudWatch log groups. |
| `cloudwatch_events_created_total` | Number of events created from processing logs from CloudWatch. |
| `api_calls_total` | Number of API calls made total. |

## Common options [filebeat-input-aws-cloudwatch-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: aws-cloudwatch
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-aws-cloudwatch-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: aws-cloudwatch
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-aws-cloudwatch]

If this option is set to true, the custom [fields](#filebeat-input-aws-cloudwatch-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


