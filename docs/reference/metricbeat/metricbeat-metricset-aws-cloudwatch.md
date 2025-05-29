---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-aws-cloudwatch.html
---

# AWS cloudwatch metricset [metricbeat-metricset-aws-cloudwatch]

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

## Fields [_fields_12]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-aws.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "aws": {
        "cloudwatch": {
            "namespace": "AWS/RDS"
        },
        "dimensions": {
            "DBClusterIdentifier": "database-1",
            "Role": "READER"
        },
        "rds": {
            "metrics": {
                "AbortedClients": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "ActiveTransactions": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "AuroraBinlogReplicaLag": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "AuroraReplicaLag": {
                    "avg": 18.4158,
                    "count": 5,
                    "max": 23.787,
                    "min": 10.634,
                    "sum": 92.07900000000001
                },
                "AuroraVolumeBytesLeftTotal": {
                    "avg": 70007366615040,
                    "count": 5,
                    "max": 70007366615040,
                    "min": 70007366615040,
                    "sum": 350036833075200
                },
                "Aurora_pq_request_attempted": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_executed": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_failed": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_in_progress": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_below_min_rows": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_few_pages_outside_buffer_pool": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_long_trx": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_pq_high_buffer_pool_pct": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_small_table": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_not_chosen_unsupported_access": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Aurora_pq_request_throttled": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "BlockedTransactions": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "BufferCacheHitRatio": {
                    "avg": 100,
                    "count": 5,
                    "max": 100,
                    "min": 100,
                    "sum": 500
                },
                "CPUUtilization": {
                    "avg": 6.051666111792592,
                    "count": 5,
                    "max": 6.216563057282379,
                    "min": 5.808333333333334,
                    "sum": 30.25833055896296
                },
                "CommitLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "CommitThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "ConnectionAttempts": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DDLLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DDLThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DMLLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DMLThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DatabaseConnections": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Deadlocks": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DeleteLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "DeleteThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "EBSByteBalance%": {
                    "avg": 99,
                    "count": 1,
                    "max": 99,
                    "min": 99,
                    "sum": 99
                },
                "EBSIOBalance%": {
                    "avg": 99,
                    "count": 1,
                    "max": 99,
                    "min": 99,
                    "sum": 99
                },
                "EngineUptime": {
                    "avg": 20800826,
                    "count": 5,
                    "max": 20800946,
                    "min": 20800706,
                    "sum": 104004130
                },
                "FreeLocalStorage": {
                    "avg": 29682751078.4,
                    "count": 5,
                    "max": 29682819072,
                    "min": 29682675712,
                    "sum": 148413755392
                },
                "FreeableMemory": {
                    "avg": 4639068160,
                    "count": 5,
                    "max": 4639838208,
                    "min": 4638638080,
                    "sum": 23195340800
                },
                "InsertLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "InsertThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "LoginFailures": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "NetworkReceiveThroughput": {
                    "avg": 0.8399323667305664,
                    "count": 5,
                    "max": 1.399556807011113,
                    "min": 0.6999533364442371,
                    "sum": 4.199661833652832
                },
                "NetworkThroughput": {
                    "avg": 1.6798647334611327,
                    "count": 5,
                    "max": 2.799113614022226,
                    "min": 1.3999066728884741,
                    "sum": 8.399323667305664
                },
                "NetworkTransmitThroughput": {
                    "avg": 0.8399323667305664,
                    "count": 5,
                    "max": 1.399556807011113,
                    "min": 0.6999533364442371,
                    "sum": 4.199661833652832
                },
                "NumBinaryLogFiles": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "Queries": {
                    "avg": 6.3836833181909265,
                    "count": 5,
                    "max": 6.53289780681288,
                    "min": 6.184260972479205,
                    "sum": 31.91841659095463
                },
                "ReadLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "ResultSetCacheHitRatio": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "RollbackSegmentHistoryListLength": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "RowLockTime": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "SelectLatency": {
                    "avg": 0.2519199153394592,
                    "count": 5,
                    "max": 0.2609050632911392,
                    "min": 0.24367924528301885,
                    "sum": 1.2595995766972958
                },
                "SelectThroughput": {
                    "avg": 2.6002296989354514,
                    "count": 5,
                    "max": 2.650618477644784,
                    "min": 2.5335866920025336,
                    "sum": 13.001148494677256
                },
                "SumBinaryLogSize": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "UpdateLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "UpdateThroughput": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                },
                "WriteLatency": {
                    "avg": 0,
                    "count": 5,
                    "max": 0,
                    "min": 0,
                    "sum": 0
                }
            }
        }
    },
    "cloud": {
        "account": {
            "id": "428152502467",
            "name": "elastic-beats"
        },
        "provider": "aws",
        "region": "eu-west-1"
    },
    "event": {
        "dataset": "aws.cloudwatch",
        "duration": 115000,
        "module": "aws"
    },
    "metricset": {
        "name": "cloudwatch",
        "period": 10000
    },
    "service": {
        "type": "aws"
    }
}
```


