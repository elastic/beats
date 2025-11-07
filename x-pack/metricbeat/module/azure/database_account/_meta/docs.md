This is the database_account metricset of the module azure.

This metricset allows users to retrieve relevant metrics from specified database accounts and comes with a predefined dashboard:

![metricbeat azure database account overview](images/metricbeat-azure-database-account-overview.png)


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_9]

`refresh_list_interval`
:   Resources will be retrieved at each fetch call (`period` interval), this means a number of Azure REST calls will be executed each time. This will be helpful if the azure users will be adding/removing resources that could match the configuration options so they will not added/removed to the list. To reduce on the number of API calls we are executing to retrieve the resources each time, users can configure this setting and make sure the list or resources will not be refreshed as often. This is also beneficial for performance and rate/ cost reasons ([https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits)).

`resources`
:   This will contain all options for identifying resources and configuring the desired metrics


### Config options to identify resources [_config_options_to_identify_resources_9]

`resource_id`
:   (*[]string*) The fully qualified ID’s of the resource, including the resource name and resource type. Has the format `/subscriptions/{{guid}}/resourceGroups/{{resource-group-name}}/providers/{{resource-provider-namespace}}/{resource-type}/{{resource-name}}`. Should return a list of resources.

`resource_group`
:   (*[]string*) This option will select all database accounts inside the resource group.

If none of the options are entered then all database accounts inside the subscription are taken in account. For each metric the primary aggregation assigned will be retrieved. A default non configurable timegrain of 5 min is set so users are advised to configure an interval of 300s or  a multiply of it.
