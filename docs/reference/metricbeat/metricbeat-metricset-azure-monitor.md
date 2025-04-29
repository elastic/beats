---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-azure-monitor.html
---

# Azure monitor metricset [metricbeat-metricset-azure-monitor]

This is the monitor metricset of the module azure.

This metricset allows users to retrieve metrics from specified resources. Added filters can apply here as the interval of retrieving these metrics, metric names, aggregation list, namespaces and metric dimensions.


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_10]

`refresh_list_interval`
:   Resources will be retrieved at each fetch call (`period` interval), this means a number of Azure REST calls will be executed each time. This will be helpful if the azure users will be adding/removing resources that could match the configuration options so they will not added/removed to the list. To reduce on the number of API calls we are executing to retrieve the resources each time, users can configure this setting and make sure the list or resources will not be refreshed as often. This is also beneficial for performance and rate/ cost reasons ([https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits)).

`resources`
:   This will contain all options for identifying resources and configuring the desired metrics


### Config options to identify resources [_config_options_to_identify_resources_10]

`resource_id`
:   (*[]string*) The fully qualified ID’s of the resource, including the resource name and resource type. Has the format `/subscriptions/{{guid}}/resourceGroups/{{resource-group-name}}/providers/{{resource-provider-namespace}}/{resource-type}/{{resource-name}}`. Should return a list of resources.

But users might have large number of resources they would like to gather metrics from so, in order to reduce the verbosity users will have the options of entering a resource group and filtering by resource type, or type in a “resource_query” where the user can filter resources from their entire subscription. Source for the resource API’s: [https://docs.microsoft.com/en-us/rest/api/resources/resources/list](https://docs.microsoft.com/en-us/rest/api/resources/resources/list) [https://docs.microsoft.com/en-us/rest/api/resources/resources/listbyresourcegroup](https://docs.microsoft.com/en-us/rest/api/resources/resources/listbyresourcegroup)

`resource_group`
:   (*[]string*) Using the resource_type configuration option as a filter is required for the resource groups entered. This option should return a list resources we want to apply our metric configuration options on.

`resource_type`
:   (*string*) As mentioned above this will be a filter option for the resource group api, will check for all resources under the specified group that are the type under this configuration.

`resource_query`
:   (*string*) Should contain a filter entered by the user, the output will be a list of resources


### Resource metric configurations [_resource_metric_configurations]

`metrics`
:   List of different metrics to collect information

`namespace`
:   (*string*) Namespaces are a way to categorize or group similar metrics together. By using namespaces, users can achieve isolation between groups of metrics that might collect different insights or performance indicators.

`name`
:   (*[]string*) Name of the metrics that’s being reported. Usually, the name is descriptive enough to help identify what’s measured. A list of metric names can be entered as well

`aggregations`
:   (*[]string*) List of supported aggregations. Azure Monitor stores all metrics at one-minute granularity intervals. During a given minute, a metric might need to be sampled several times or it might need to be measured for many discrete events. To limit the number of raw values we have to emit and pay for in Azure Monitor, they will locally pre-aggregate and emit the values: Minimum: The minimum observed value from all the samples and measurements during the minute. Maximum: The maximum observed value from all the samples and measurements during the minute. Sum: The summation of all the observed values from all the samples and measurements during the minute. Count: The number of samples and measurements taken during the minute. Total: The total number of all the observed values from all the samples and measurements during the minute.

`dimensions`
:   List of metric dimensions. Dimensions are optional, not all metrics may have dimensions. A custom metric can have up to 10 dimensions. A dimension is a key or value pair that helps describe additional characteristics about the metric being collected. By using the additional characteristics, you can collect more information about the metric, which allows for deeper insights. By using this key, you can filter the metric to see how much memory specific processes use or to identify the top five processes by memory usage. Metrics with dimensions are exported as flattened single dimensional metrics, aggregated across dimension values.

`name`
:   Dimension key

`value`
:   Dimension value. (Users can select * to return metric values for each dimension)

`ignore_unsupported`
:   (*bool*) Namespaces can be unsupported by some resources and supported in some, this configuration option makes sure no error messages are returned if the namespace is unsupported. The same will go for the metrics configured, some can be removed from Azure Monitor and it should not affect the state of the module.

Users can select the options to retrieve all metrics from a specific namespace using the following:

```yaml
 metrics:
 - name: ["*"]
   namespace: "Microsoft.Storage/storageAccounts"
```

If no aggregations are entered under a metric level the metricset will retrieve the primary aggregation assigned for this metric.

This is a default metricset. If the host module is unconfigured, this metricset is enabled by default.

## Fields [_fields_38]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-azure.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "azure": {
        "dimensions": {
            "activity_name": "secretlist"
        },
        "metrics": {
            "availability": {
                "avg": 100
            }
        },
        "namespace": "Microsoft.KeyVault/vaults",
        "resource": {
            "group": "some-rg",
            "id": "/subscriptions/70f046a0-a299-ab73-9950-88ac8b5ac454/resourceGroups/some-rg/providers/Microsoft.KeyVault/vaults/somekeyvault",
            "name": "somekeyvault",
            "type": "Microsoft.KeyVault/vaults"
        },
        "subscription_id": "70bd6e64-4b1e-4835-8896-db77b8eef364",
        "timegrain": "PT1M"
    },
    "cloud": {
        "provider": "azure",
        "region": "westeurope"
    },
    "event": {
        "dataset": "azure.monitor",
        "duration": 115000,
        "module": "azure"
    },
    "metricset": {
        "name": "monitor",
        "period": 10000
    },
    "service": {
        "type": "azure"
    }
}
```


