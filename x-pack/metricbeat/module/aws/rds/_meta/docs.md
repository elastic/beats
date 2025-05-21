The rds metricset of aws module allows you to monitor your AWS RDS service. `rds` metricset fetches a set of metrics from [Amazon RDS](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/MonitoringOverview.html) and [Amazon Aurora DB](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Aurora.Monitoring.html). with Amazon RDS, users can monitor network throughput, I/O for read, write, and/or metadata operations, client connections, and burst credit balances for their DB instances. Amazon RDS sends metrics and dimensions to Amazon CloudWatch every minute. Amazon Aurora provides a variety of Amazon CloudWatch metrics that users can use to monitor health and performance of their Aurora DB cluster. This metricset by default collects all tags from AWS RDS.


## AWS Permissions [_aws_permissions_10]

Some specific AWS permissions are required for IAM user to collect AWS RDS metrics.

```
cloudwatch:GetMetricData
ec2:DescribeRegions
rds:DescribeDBInstances
rds:ListTagsForResource
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_11]

The aws rds metricset comes with a predefined dashboard.

![metricbeat aws rds overview](images/metricbeat-aws-rds-overview.png)


## Configuration example [_configuration_example_10]

```yaml
- module: aws
  period: 60s
  metricsets:
    - rds
  access_key_id: '<access_key_id>'
  secret_access_key: '<secret_access_key>'
  session_token: '<session_token>'
```

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
