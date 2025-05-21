The cloudwatch metricset of aws module allows you to monitor various services on AWS. `cloudwatch` metricset fetches metrics from given namespace periodically by calling `GetMetricData` api.


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

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.
