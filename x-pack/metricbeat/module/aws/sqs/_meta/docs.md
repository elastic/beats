The sqs metricset of aws module allows you to monitor your AWS SQS queues. `sqs` metricset fetches a set of values from [Amazon SQS Metrics](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-available-cloudwatch-metrics.html). CloudWatch metrics for Amazon SQS queues are automatically collected and pushed to CloudWatch every five minutes.


## AWS Permissions [_aws_permissions_14]

Some specific AWS permissions are required for IAM user to collect AWS SQS metrics.

```
cloudwatch:GetMetricData
cloudwatch:ListMetrics
ec2:DescribeRegions
sqs:ListQueues
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_15]

The aws sqs metricset comes with a predefined dashboard. For example:

![metricbeat aws sqs overview](images/metricbeat-aws-sqs-overview.png)


## Configuration example [_configuration_example_14]

```yaml
- module: aws
  period: 300s
  metricsets:
    - sqs
  access_key_id: '<access_key_id>'
  secret_access_key: '<secret_access_key>'
  session_token: '<session_token>'
```
