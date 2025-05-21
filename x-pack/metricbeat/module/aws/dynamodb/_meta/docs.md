::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `dynamodb` metricset of aws module allows you to monitor your AWS DynamoDB database. `dynamodb` metricset fetches a set of values from [Amazon DynamoDB Metrics](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/metrics-dimensions.html). For all other DynamoDB metrics, the aggregation granularity is five minutes.


## Configuration example [_configuration_example_3]

```yaml
- module: aws
  period: 300s
  metricsets:
    - dynamodb
  # This module uses the aws cloudwatch metricset, all
  # the options for this metricset are also available here.
```


## Dashboard [_dashboard_4]

The aws dynamodb metricset comes with a predefined dashboard. For example:

![metricbeat aws dynamodb overview](images/metricbeat-aws-dynamodb-overview.png)


## Metrics [_metrics_2]

Please see more details for each metric in [Amazon DynamoDB Metrics](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/metrics-dimensions.html).

| Metric Name | Statistic Method |
| --- | --- |
| SuccessfulRequestLatency | Average |
| OnlineIndexPercentageProgress | Average |
| ProvisionedWriteCapacityUnits | Average |
| ProvisionedReadCapacityUnits | Average |
| ConsumedReadCapacityUnits | Average |
| ConsumedWriteCapacityUnits | Average |
| ReplicationLatency | Average |
| TransactionConflict | Average |
| AccountProvisionedReadCapacityUtilization | Average |
| AccountProvisionedWriteCapacityUtilization | Average |
| SystemErrors | Sum |
| ConsumedReadCapacityUnits | Sum |
| ConsumedWriteCapacityUnits | Sum |
| ConditionalCheckFailedRequests | Sum |
| PendingReplicationCount | Sum |
| TransactionConflict | Sum |
| ReadThrottleEvents | Sum |
| ThrottledRequests | Sum |
| WriteThrottleEvents | Sum |
| SuccessfulRequestLatency | Maximum |
| ReplicationLatency | Maximum |
| AccountMaxReads | Maximum |
| AccountMaxTableLevelReads | Maximum |
| AccountMaxTableLevelWrites | Maximum |
| AccountMaxWrites | Maximum |
| MaxProvisionedTableReadCapacityUtilization | Maximum |
| MaxProvisionedTableWriteCapacityUtilization | Maximum |

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
