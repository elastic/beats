Amazon Elastic Block Store (Amazon EBS) sends data points to CloudWatch for several metrics. Most EBS volumes automatically send five-minute metrics to CloudWatch only when the volume is attached to an instance. This aws `ebs` metricset collects these Cloudwatch metrics for monitoring purposes.


## AWS Permissions [_aws_permissions_4]

Some specific AWS permissions are required for IAM user to collect AWS EBS metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_5]

The aws ebs metricset comes with a predefined dashboard. For example:

![metricbeat aws ebs overview](images/metricbeat-aws-ebs-overview.png)


## Configuration example [_configuration_example_4]

```yaml
- module: aws
  period: 300s
  metricsets:
    - ebs
  # This module uses the aws cloudwatch metricset, all
  # the options for this metricset are also available here.
```


## Metrics [_metrics_3]

Please see more details for each metric in [ebs-cloudwatch-metric](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using_cloudwatch_ebs.html).

| Metric Name | Statistic Method |
| --- | --- |
| VolumeReadBytes | Average |
| VolumeWriteBytes | Average |
| VolumeReadOps | Average |
| VolumeWriteOps | Average |
| VolumeQueueLength | Average |
| VolumeThroughputPercentage | Average |
| VolumeConsumedReadWriteOps | Average |
| BurstBalance | Average |
| VolumeTotalReadTime | Sum |
| VolumeTotalWriteTime | Sum |
| VolumeIdleTime | Sum |
