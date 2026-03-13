::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the app_insights metricset.

This metricset allows users to retrieve application insights metrics from specified applications.


### Config options to identify resources [_config_options_to_identify_resources]

`application_id`
:   (*string*) ID of the application. This is Application ID from the API Access settings blade in the Azure portal.


### Authentication [_authentication]

Two authentication methods are supported: **Client secret (Microsoft Entra ID)** and **API key**. The method is selected using the `auth_type` option.

`auth_type`
:   (*string*) The authentication method to use. Valid values: `api_key` (default), `client_secret`.

#### Client secret authentication

{applies_to}`stack: ga 8.19.13` {applies_to}`stack: ga 9.2.7` {applies_to}`stack: ga 9.3.2`

Set `auth_type: "client_secret"` and provide the following options:

`tenant_id`
:   (*string*) The tenant ID of the Microsoft Entra ID (Azure Active Directory) instance. More on service principal authentication can be found here [https://learn.microsoft.com/en-us/entra/identity-platform/howto-create-service-principal-portal](https://learn.microsoft.com/en-us/entra/identity-platform/howto-create-service-principal-portal).

`client_id`
:   (*string*) The client/application ID of the service principal registered in Microsoft Entra ID.

`client_secret`
:   (*string*) The client secret associated with the service principal.

All three of `tenant_id`, `client_id`, and `client_secret` are required when `auth_type` is `client_secret`.

**Required permissions:** The service principal must be assigned a role that grants read access to Application Insights data. The minimum built-in role is **Monitoring Reader**, assigned at the Application Insights resource scope. Other roles that include the required permissions are **Monitoring Contributor**, **Contributor**, and **Owner**. For more details, see [Azure built-in roles for Monitor](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/monitor).

#### API key authentication

::::{warning}
Microsoft is retiring API key authentication for Application Insights on **March 31, 2026**. After this date, API key authentication will no longer work. It is recommended to migrate to [client secret authentication](#_authentication) before this deadline. For more details, see [Transition to Microsoft Entra ID authentication](https://azure.microsoft.com/en-us/updates?id=transition-to-azure-ad-to-query-data-from-azure-monitor-application-insights-by-31-march-2026).
::::

Set `auth_type: "api_key"` (or omit `auth_type`, as it defaults to `api_key`) and provide:

`api_key`
:   (*string*) The API key which will be generated, more on the steps here [https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID](https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID).

**Required permissions:** The API key must be created with the **Read telemetry** permission enabled in the Azure portal (under the API Access blade of the Application Insights resource).


### App insights metric configurations [_app_insights_metric_configurations]

`metrics`
:   List of different metrics to collect information

`id`
:   (*[]string*) IDs of the metrics that’s being reported. Usually, the id is descriptive enough to help identify what’s measured. A list of metric names can be entered as well. Default metricsets include: `requests/count` `requests/duration` `requests/failed` `users/count``users/authenticated` `pageViews/count` `pageViews/duration` `customEvents/count` `browserTimings/processingDuration` `browserTimings/receiveDuration` `browserTimings/networkDuration` `browserTimings/sendDuration` `browserTimings/totalDuration` `dependencies/count` `dependencies/duration` `dependencies/failed` `exceptions/count` `exceptions/browser` `exceptions/server` `sessions/count` `performanceCounters/requestExecutionTime` `performanceCounters/requestsPerSecond` `performanceCounters/requestsInQueue` `performanceCounters/memoryAvailableBytes` `performanceCounters/exceptionsPerSecond` `performanceCounters/processCpuPercentage` `performanceCounters/processIOBytesPerSecond` `performanceCounters/processPrivateBytes` `performanceCounters/processorCpuPercentage` `availabilityResults/count` `availabilityResults/availabilityPercentage` `availabilityResults/duration`

`interval`
:   (*string*) The time interval to use when retrieving metric values. This is an ISO8601 duration. If interval is omitted, the metric value is aggregated across the entire timespan. If interval is supplied, the result may adjust the interval to a more appropriate size based on the timespan used for the query.

`aggregation`
:   (*[]string*) The aggregation to use when computing the metric values. To retrieve more than one aggregation at a time, separate them with a comma. If no aggregation is specified, then the default aggregation for the metric is used.

`segment`
:   (*[]string*) The name of the dimension to segment the metric values by. This dimension must be applicable to the metric you are retrieving. In this case, the metric data will be segmented in the order the dimensions are listed in the parameter.

`top`
:   (*int*) The number of segments to return. This value is only valid when segment is specified.

`order_by`
:   (*string*) The aggregation function and direction to sort the segments by. This value is only valid when segment is specified.

`filter`
:   (*string*) An expression used to filter the results. This value should be a valid OData filter expression where the keys of each clause should be applicable dimensions for the metric you are retrieving.

Example configuration:

```yaml
metrics:
 - id: ["requests/count", "requests/failed"]
   segment: "request/name"
   aggregation: ["sum"]
```
