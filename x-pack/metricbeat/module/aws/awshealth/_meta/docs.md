::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


AWS Health metrics provide insights into the health of your AWS environment by monitoring various aspects such as open issues, scheduled maintenance events, security advisories, compliance status, notification counts, and service disruptions. These metrics help you proactively identify and address issues impacting your AWS resources, ensuring the reliability, security, and compliance of your infrastructure.


## AWS Permissions [_aws_permissions]

To collect AWS Health metrics using Elastic Metricbeat, you would need specific AWS permissions to access the necessary data. Here’s a list of permissions required for an IAM user to collect AWS Health metrics:

```
health:DescribeAffectedEntities
health:DescribeEventDetails
health:DescribeEvents
```


## Configuration example [_configuration_example]

```yaml
- module: aws
  period: 24h
  metricsets:
    - awshealth
```
