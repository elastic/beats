::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the app_insights metricset.

This metricset allows users to retrieve application insights metrics from specified applications.


### Config options to identify resources [_config_options_to_identify_resources]

`application_id`
:   (*[]string*) ID of the application. This is Application ID from the API Access settings blade in the Azure portal.

`api_key`
:   (*[]string*) The API key which will be generated, more on the steps here [https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID](https://dev.applicationinsights.io/documentation/Authorization/API-key-and-App-ID).


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
