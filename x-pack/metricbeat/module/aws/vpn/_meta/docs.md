::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The vpn metricset of aws module allows users to monitor VPN tunnels. VPN metric data is automatically sent to CloudWatch as it becomes available. Users can use these metrics to gain a better perspective on how the web application or service is performing.


## AWS Permissions [_aws_permissions_17]

Some specific AWS permissions are required for IAM user to collect usage metrics.

```
ec2:DescribeRegions
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
```


## Dashboard [_dashboard_18]

The aws vpn metricset comes with a predefined dashboard. For example:

![metricbeat aws vpn overview](images/metricbeat-aws-vpn-overview.png)


## Configuration example [_configuration_example_17]

```yaml
- module: aws
  period: 1m
  metricsets:
    - vpn
  # This module uses the aws cloudwatch metricset, all
  # the options for this metricset are also available here.
```


## Metrics and Dimensions for VPN [_metrics_and_dimensions_for_vpn]

Metrics:

| Metric Name | Statistic Method | Description |
| --- | --- | --- |
| TunnelState | Max | The state of the tunnel. For static VPNs, 0 indicates DOWN and 1 indicates UP. For BGP VPNs, 1 indicates ESTABLISHED and 0 is used for all other states. |
| TunnelDataIn | Sum | The bytes received through the VPN tunnel. |
| TunnelDataOut | Sum | The bytes sent through the VPN tunnel. |

Dimensions:

| Dimension Name | Description |
| --- | --- |
| VpnId | Filters the metric data by the Site-to-Site VPN connection ID. |
| TunnelIpAddress | Filters the metric data by the IP address of the tunnel for the virtual private gateway. |

Please see [VPN Tunnel CloudWatch Metrics](https://docs.aws.amazon.com/vpn/latest/s2svpn/monitoring-cloudwatch-vpn.html) for more details.
