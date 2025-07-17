::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the app_state metricset of the module azure.

This metricset allows users to retrieve application insights metrics from specified applications.


### Config options to identify resources [_config_options_to_identify_resources_2]

`application_id`
:   (*[]string*) ID of the application. This is Application ID from the API Access settings blade in the Azure portal.

`api_key`
:   (*[]string*) The API key which will be generated, more on the steps here [https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID](https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID).
