::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The transitgateway metricset of aws module allows users to monitor transit gateway. Transit gateway metrics are sent to CloudWatch by VPC only when requests are flowing through the gateway. If there are requests flowing through the transit gateway, Amazon VPC measures and sends its metrics in 1-minute intervals. Users can use these metrics to gain a better perspective on how the web application or service is performing.


## AWS Permissions [_aws_permissions_15]

Some specific AWS permissions are required for IAM user to collect usage metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_16]

The aws transitgateway metricset comes with a predefined dashboard. For example:

![metricbeat aws transitgateway overview](images/metricbeat-aws-transitgateway-overview.png)


## Configuration example [_configuration_example_15]

```yaml
- module: aws
  period: 1m
  metricsets:
    - transitgateway
  # This module uses the aws cloudwatch metricset, all
  # the options for this metricset are also available here.
```


## Metrics and Dimensions for Transit gateway [_metrics_and_dimensions_for_transit_gateway]

Metrics:

| Metric Name | Statistic Method | Description |
| --- | --- | --- |
| BytesIn | Sum | The number of bytes received by the transit gateway. |
| BytesOut | Sum | The number of bytes sent from the transit gateway. |
| PacketsIn | Sum | The number of packets received by the transit gateway. |
| PacketsOut | Sum | The number of packets sent by the transit gateway. |
| PacketDropCountBlackhole | Sum | The number of packets dropped because they matched a blackhole route. |
| PacketDropCountNoRoute | Sum | The number of packets dropped because they did not match a route. |

Dimensions:

| Dimension Name | Description |
| --- | --- |
| TransitGateway | Filters the metric data by transit gateway. |

Please see [Transit Gateway Metrics](https://docs.aws.amazon.com/vpc/latest/tgw/transit-gateway-cloudwatch-metrics.html) for more details.
