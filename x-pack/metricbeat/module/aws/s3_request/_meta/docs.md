The s3_request metricset of aws module allows you to monitor your AWS S3 buckets. `s3_request` metricset fetches Cloudwatch daily storage metrics for each S3 bucket from [S3 CloudWatch Request Metrics for Buckets](https://docs.aws.amazon.com/AmazonS3/latest/dev/cloudwatch-monitoring.html).

Note: Request metrics are not enabled by default. You must opt into request metrics by configuring them in the console or using the Amazon S3 API. Please see [How to Configure Request Metrics for S3](https://docs.aws.amazon.com/AmazonS3/latest/user-guide/configure-metrics.html) for instructions on how to enable request metrics for each S3 bucket.


## AWS Permissions [_aws_permissions_12]

Some specific AWS permissions are required for IAM user to collect AWS s3_request metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_13]

The aws s3_request metricset and s3_daily_storage metricset shares one predefined dashboard. For example:

![metricbeat aws s3 overview](images/metricbeat-aws-s3-overview.png)

Note: If s3 request metrics are not enabled for the specific S3 bucket, some visualizations in this dashboard will be empty.


## Configuration example [_configuration_example_12]

```yaml
- module: aws
  period: 86400s
  metricsets:
    - s3_request
  access_key_id: '<access_key_id>'
  secret_access_key: '<secret_access_key>'
  session_token: '<session_token>'
```
