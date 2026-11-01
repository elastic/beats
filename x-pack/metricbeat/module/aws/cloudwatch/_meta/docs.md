The cloudwatch metricset of aws module allows you to monitor various services on AWS. `cloudwatch` metricset fetches metrics from given namespace periodically by calling `GetMetricData` api.



## Internal collection metrics

Each running metricset instance exposes internal metrics through Metricbeat's
`/inputs/` monitoring endpoint. Agent diagnostics can include the same records in
`input_metrics.json` when diagnostics routing is available. Records retain the
configured `id` and a unique `ephemeral_id`; instances sharing a configured ID
are reported separately. Counters reset when the instance is recreated.

The shared Metricbeat wrapper provides:

| Metric | Meaning |
| --- | --- |
| `success`, `failures` | Events reported without and with an error, respectively. These are not fetch counts. |
| `events` | Events handed to the publishing pipeline, including error events. This does not imply output acknowledgment. |
| `fetches_total` | Finished periodic fetch invocations, including failures. |
| `fetch_errors_total` | Fetch invocations returning a non-partial error or unwinding after a panic. |
| `fetch_partial_errors_total` | Fetch invocations returning a partial metrics error. |
| `consecutive_failures` | Consecutive non-partial fetch errors for this instance; reset by a successful or partial fetch. |
| `fetch_in_progress` | Whether a periodic fetch is running, including time spent reporting its events. |
| `fetch_started_at`, `last_fetch_completed_at` | Unix timestamps in milliseconds; zero until observed. |
| `fetch_duration_ns` | Sampled duration of completed fetches, including event handoff. |
| `publish_wait_ns` | Sampled time spent handing an event to the wrapper's output channel, including canceled handoffs. |

Duration histograms use bounded uniform reservoirs of 256 observations. Push
metricsets expose event metrics but do not increment periodic fetch metrics.
Fetch errors reported only through events, or logged without being returned,
do not increment `fetch_errors_total`.

CloudWatch additionally exposes a `cloudwatch` object:

| Metric | Meaning |
| --- | --- |
| `period_ms`, `latency_ms`, `data_granularity_ms`, `include_linked_accounts` | Effective collection settings. |
| `window_start`, `window_end` | Most recently attempted query window, as Unix timestamps in milliseconds. |
| `last_successful_window_end` | End of the last fetch window completed without observed collection errors, final SDK errors, or incomplete AWS results. Absent until observed. This is not an output acknowledgment. |
| `collection_lag_ms` | Distance from the attempted window end to the expected latency-adjusted, period-aligned window end, computed when metrics are read. |
| `successful_collection_lag_ms` | The same distance measured from the last successful window; absent until one succeeds. |
| `collection_errors_total` | Errors observed at discovery/collection stages or returned from the fetch. |
| `incomplete_fetches_total` | Fetches with an observed error or incomplete AWS result, even when `Fetch` returns no error. |
| `incomplete_results_total` | Metric data results with a non-complete AWS status. |
| `metric_queries_total`, `datapoints_received_total` | Queries submitted for retrieval and datapoints returned, including partial results. These are distinct from the number of created events. |

Lag excludes configured latency and window alignment. It continues increasing
while a fetch is blocked. The attempted cursor advances before collection
completes, so use the successful-window metrics together with error counters.
These metrics do not change cursor advancement or catch-up behavior.

`cloudwatch.scopes` contains bounded records identified by `region` and
`namespace`, including configured namespaces that have never produced events.
Each record includes discovery calls/errors, cumulative and most recent metrics
discovered, collection calls/errors, empty collections, created events, cumulative
discovery/collection duration in nanoseconds, and `last_success_at` in Unix
milliseconds. The last success refers to a collection stage completing without a
returned error. Consult incomplete-result metrics separately. A `*` record groups
wildcard discovery, dynamically discovered namespaces, or explicit metric queries
that are collected together. Scope duration excludes event handoff; metadata
enrichment is included where performed.

`cloudwatch.api` contains bounded operation records for CloudWatch, tagging,
API Gateway, and resource metadata calls. Each record exposes `calls_total`,
`errors_total`, `throttles_total`, and sampled duration count, maximum, mean, and
95th percentile in nanoseconds. An SDK invocation counts once per page, including
all retry time. Errors and throttles count final failures; successful calls that
previously retried are not reported as failures. Unlisted operations are grouped
under `Other`. Credential setup and initial region/account discovery are outside
this instrumentation.

Compare fetch duration with `period_ms`, namespace stage durations with each other,
and API duration with publication wait to distinguish expensive collection from
publishing backpressure. Publication wait measures a local handoff, not the full
Elasticsearch request duration. Cross-account work is aggregated; these metrics
do not enumerate linked accounts or resources.

## AWS Permissions [_aws_permissions_3]

Some specific AWS permissions are required for IAM user to collect AWS Cloudwatch metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_2]

* **namespace**: The namespace used by ListMetrics API to filter against. For example, AWS/EC2, AWS/S3. If wildcard * is given for namespace, metrics from all namespaces will be collected automatically.
* **name**: The name of the metric to filter against. For example, CPUUtilization for EC2 instance.
* **dimensions**: The dimensions to filter against. For example, InstanceId=i-123.
* **resource_type**: The constraints on the resources that you want returned. The format of each resource type is service[:resourceType]. For example, specifying a resource type of ec2 returns all Amazon EC2 resources (which includes EC2 instances). Specifying a resource type of ec2:instance returns only EC2 instances.
* **statistic**: Statistics are metric data aggregations over specified periods of time. By default, statistic includes Average, Sum, Count, Maximum and Minimum.


## Configuration examples [_configuration_examples]

To be more focused on `cloudwatch` metricset use cases, the examples below do not include configurations on AWS credentials. Please see [AWS credentials options](/reference/metricbeat/metricbeat-module-aws.md#aws-credentials-config) for more details on setting AWS credentials in configurations in order for this metricset to make proper AWS API calls.


### Example 1 [_example_1]

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  tags_filter: <3>
    - key: "Organization"
      value: "Engineering"
  metrics:
    - namespace: AWS/EBS <1>
    - namespace: AWS/ELB <2>
      resource_type: elasticloadbalancing
    - namespace: AWS/EC2 <4>
      name: CPUUtilization
      statistic: ["Average"]
      dimensions:
        - name: InstanceId
          value: i-0686946e22cf9494a
```

1. Users can configure the `cloudwatch` metricset to collect all metrics from one specific namespace, such as `AWS/EBS`.
2. `cloudwatch` metricset also has the ability to collect tags from AWS resources. If `resource_type` is specified, then tags will be collected and stored as a part of the event. Please see [AWS API GetResources](https://docs.aws.amazon.com/resourcegroupstagging/latest/APIReference/API_GetResources.html) for more details about `resource_type`.
3. If tags are collected (for metricsets with `resource_type` specified), events can also be filtered by tag, using the `tags_filter` field in the module-specific configuration.
4. If users knows exactly what are the cloudwatch metrics they want to collect, this configuration format can be used. `namespace` and `metricname` need to be specified and `dimensions` can be used to filter cloudwatch metrics. Please see [AWS List Metrics](https://docs.aws.amazon.com/cli/latest/reference/cloudwatch/list-metrics.html) for more details.



### Example 2 [_example_2]

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  metrics:
    - namespace: "*"
```

With this config, metrics from all namespaces will be collected from Cloudwatch. The limitation here is the collection period for all namespaces are all set to be the same, which in this case is 300 second. This will cause extra costs for API calls or data loss. For example, metrics from namespace AWS/Usage are sent to Cloudwatch every 1 minute. With the collection period equals to 300 seconds, data points in between will get lost. Metrics from namespace AWS/Billing are sent to Cloudwatch every several hours. By querying from AWS/Billing namespace every 300 seconds, additional costs will occur.


### Example 3 [_example_3]

Depends on the configuration and number of services in the AWS account, the number of API calls may get too big to cause high API cost. In order to reduce the number of API calls, we recommend users to use this configuration below as an example.

* **metrics.name**: Only collect a sub list of metrics that are useful to your use case.
* **metrics.statistic**: By default, cloudwatch metricset will make API calls to get all stats like average, max, min, sum and etc. If the user knows which statistics method is most useful, specify it in the configuration.
* **metrics.dimensions**: Different AWS services report different dimensions in their CloudWatch metrics. For example, [EMR metrics](https://docs.aws.amazon.com/emr/latest/ManagementGuide/UsingEMR_ViewingMetrics.html) can have either `JobFlowId` dimension or `JobId` dimension. If user knows which specific dimension is useful, it can be specified in this configuration option.

```yaml
- module: aws
  period: 5m
  metricsets:
    - cloudwatch
  regions: us-east-1
  metrics:
    - namespace: AWS/ElasticMapReduce
      name: ["S3BytesWritten", "S3BytesRead", "HDFSUtilization", "TotalLoad"]
      resource_type: elasticmapreduce
      statistic: ["Average"]
      dimensions:
        - name: JobId
          value: "*"
```


## More examples [_more_examples]

With the configuration below, users will be able to collect cloudwatch metrics from EBS, ELB and EC2 without tag information.

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  metrics:
    - namespace: AWS/EBS
    - namespace: AWS/ELB
    - namespace: AWS/EC2
```

With the configuration below, users will be able to collect cloudwatch metrics from EBS, ELB and EC2 with tags from these services.

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  metrics:
    - namespace: AWS/EBS
      resource_type: ebs
    - namespace: AWS/ELB
      resource_type: elasticloadbalancing
    - namespace: AWS/EC2
      resource_type: ec2:instance
```

With the configuration below, users will be able to collect specific cloudwatch metrics. For example CPUUtilization metric(average) from EC2 instance i-123 and NetworkIn metric(average) from EC2 instance i-456.

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  metrics:
    - namespace: AWS/EC2
      name: ["CPUUtilization"]
      resource_type: ec2:instance
      dimensions:
        - name: InstanceId
          value: i-123
      statistic: ["Average"]
    - namespace: AWS/EC2
      name: ["NetworkIn"]
      dimensions:
        - name: InstanceId
          value: i-456
      statistic: ["Average"]
```

With the configuration below, user can filter out only `LoadBalacer` and `TargetGroup` dimension metircs with the metric name `UnHealthyHostCount`, `LoadBalacer` and `TargetGroup` value could be any.

```yaml
- module: aws
  period: 300s
  metricsets:
    - cloudwatch
  metrics:
    - namespace: AWS/ApplicationELB
      statistic: ['Maximum']
      name: ['UnHealthyHostCount']
      dimensions:
        - name: LoadBalancer
          value: "*"
        - name: TargetGroup
          value: "*"
      resource_type: elasticloadbalancing
```
