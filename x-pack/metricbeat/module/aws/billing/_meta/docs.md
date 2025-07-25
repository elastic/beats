::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


You can monitor your estimated AWS charges by using Amazon CloudWatch and Cost Explorer.

This aws `billing` metricset collects metrics both from Cloudwatch and cost explorer for monitoring purposes.


## AWS Permissions [_aws_permissions_2]

Some specific AWS permissions are required for IAM user to collect estimated billing metrics.

```
cloudwatch:GetMetricData
cloudwatch:ListMetrics
tag:getResources
sts:GetCallerIdentity
iam:ListAccountAliases
ce:GetCostAndUsage
organizations:ListAccounts
```


## Dashboard [_dashboard_3]

The aws billing metricset comes with a predefined dashboard. For example:

![metricbeat aws billing overview](images/metricbeat-aws-billing-overview.png)


## Configuration example [_configuration_example_2]

```yaml
- module: aws
  period: 24h
  metricsets:
    - billing
  credential_profile_name: elastic-beats
  cost_explorer_config:
    group_by_dimension_keys:
      - "AZ"
      - "INSTANCE_TYPE"
      - "SERVICE"
    group_by_tag_keys:
      - "aws:createdBy"
```


## Metricset-specific configuration notes [_metricset_specific_configuration_notes]

When querying AWS Cost Explorer API, you can group AWS costs using up to two different groups, either dimensions, tag keys, or both. Right now we support group by type dimension and type tag with separate config parameters:

* **group_by_dimension_keys**: A list of keys used in Cost Explorer to group by dimensions. Valid values are AZ, INSTANCE_TYPE, LINKED_ACCOUNT, OPERATION, PURCHASE_TYPE, REGION, SERVICE, USAGE_TYPE, USAGE_TYPE_GROUP, RECORD_TYPE, OPERATING_SYSTEM, TENANCY, SCOPE, PLATFORM, SUBSCRIPTION_ID, LEGAL_ENTITY_NAME, DEPLOYMENT_OPTION, DATABASE_ENGINE, CACHE_ENGINE, INSTANCE_TYPE_FAMILY, BILLING_ENTITY and RESERVATION_ID.
* **group_by_tag_keys**: A list of keys used in Cost Explorer to group by tags.
