AWS Lambda monitors functions and sends metrics to Amazon CloudWatch. These metrics include total invocations, errors, duration, throttles, dead-letter queue errors, and iterator age for stream-based invocations.


## AWS Permissions [_aws_permissions_8]

Some specific AWS permissions are required for IAM user to collect AWS EBS metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_9]

The aws lambda metricset comes with a predefined dashboard. For example:

![metricbeat aws lambda overview](images/metricbeat-aws-lambda-overview.png)


## Configuration example [_configuration_example_8]

```yaml
- module: aws
  period: 300s
  metricsets:
    - lambda
  # This module uses the aws cloudwatch metricset, all
  # the options for this metricset are also available here.
```


## Metrics [_metrics_5]

Please see more details for each metric in [lambda-cloudwatch-metric](https://docs.aws.amazon.com/lambda/latest/dg/monitoring-functions-metrics.html).

| Metric Name | Statistic Method |
| --- | --- |
| Invocations | Average |
| Errors | Average |
| DeadLetterErrors | Average |
| DestinationDeliveryFailures | Average |
| Duration | Average |
| Throttles | Average |
| IteratorAge | Average |
| ConcurrentExecutions | Average |
| UnreservedConcurrentExecutions | Average |
| ProvisionedConcurrentExecutions | Maximum |
| ProvisionedConcurrencyInvocations | Sum |
| ProvisionedConcurrencySpilloverInvocations | Sum |
| ProvisionedConcurrencyUtilization | Maximum |
