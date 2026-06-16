::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the billing metricset of the module azure.

This metricset allows users to retrieve usage details and forecast information of the subscription configured.


## Metricset-specific configuration notes [_metricset_specific_configuration_notes_3]

`refresh_list_interval`
:   Resources will be retrieved at each fetch call (`period` interval), this means a number of Azure REST calls will be executed each time. This will be helpful if the azure users will be adding/removing resources that could match the configuration options so they will not added/removed to the list. To reduce on the number of API calls we are executing to retrieve the resources each time, users can configure this setting and make sure the list or resources will not be refreshed as often. This is also beneficial for performance and rate/ cost reasons ([https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-request-limits)).

`resources`
:   This will contain all options for identifying resources and configuring the desired metrics


### Config options to identify resources [_config_options_to_identify_resources_3]

`billing_scope_department`
:   (*string*) Retrieve usage details based on the department scope.

`billing_scope_account_id`
:   (*string*) Retrieve usage details based on the billing account ID scope.

If none of the 2 options are entered then the subscription ID will be used as scope.
