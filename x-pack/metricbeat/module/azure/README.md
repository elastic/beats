### Azure Monitor Walkthrough when EnableBatchApi is True

#### Initialization Phase

1.  **InitResources Method**:
    
    -   **Validate Resources**: Checks if any resources are defined in the user configuration.
        
    -   **Check Refresh Interval**: If the refresh interval is not expired, it initializes the `MetricDefinitionsChan` and `ErrorChan` channels and sends existing metric definitions through the channel. These metric definitions have been collected in previous collection times.
        
    -   **Initialize WaitGroup**: Creates a `sync.WaitGroup` to track all goroutines for resource collection.
        
    -   **Retrieve Resource Definitions**: Iterates over user-configured resources and retrieves their definitions from Azure Monitor.
        
    -   **Check Resource Definitions**: If, for a given user configuration, no resources have been retrieved, an error is logged, and it continues to the next resource of the configuration.
        
    -   **Initialization of Channels**: `MetricDefinitionsChan` and `ErrorChan` are initialized once. The `MetricDefinitionsChan` channel will be used to receive the metric definitions of all resources of the provided configuration. `ErrorChan` will be used to report errors in the metric definitions collection process.
        
    -   **Map Resources to Client**: Maps the retrieved resources to the client's resource list.
        
    -   **Collect Metric Definitions**: For each resource, calls the provided mapping function (`mapMetrics`) to collect metric definitions. Refer to the **mapMetrics Function**.
        
    -   **Close Channels**: Once all goroutines complete, it closes the `MetricDefinitionsChan` and `ErrorChan` channels. This signals that all metric definitions of all resources in the configuration are collected.
        
2.  **mapMetrics Function**:
    
    -   **Start Goroutine**: Starts a new goroutine for each resource to collect its metric definitions.
        
    -   **Retrieve Metric Definitions**: Calls `getMappedResourceDefinitions` to retrieve and map metric definitions for each resource. Refer to the **getMappedResourceDefinitions Function**.
        
    -   **Check for Errors**: In case `getMappedResourceDefinitions` returns an error, it is sent to the `ErrorChan`. This will cause the data collection to stop.
        
    -   **Send to Channel**: Sends the retrieved metric definitions to the `MetricDefinitionsChan` channel.
        
3.  **getMappedResourceDefinitions Function**:
    
    -   **Avoid Redundant Calls**: Uses a map to avoid calling the metric definitions function multiple times for the same namespace and resource.
        
    -   **Retrieve Metric Definitions**: Retrieves metric definitions from Azure Monitor for the specified resource.
        
    -   **Filter Supported Metrics**: Validates and filters the metric names and aggregations based on the supported metrics.
        
    -   **Map Dimensions**: Maps dimensions to the metrics as specified in the resource configuration.
        
    -   **Return Metrics**: Returns the list of mapped metrics.
        

#### Data Collection and Processing Phase

4.  **Fetch Method**:
    
    -   **Set Reference Time**: The `Fetch` method starts by setting the reference time for the current fetch operation. This is used to calculate time intervals for metrics collection.
        
    -   **Initialize Resources**: It calls the `InitResources` method to collect and validate resources based on user configuration. Refer to the Initialization Phase.
        
    -   **Check Channel Initialization**: If the `MetricDefinitionsChan` channel is `nil`, it returns an error, indicating no resources were found based on the configurations.
        
    -   **Create Metric Stores**: Initializes a map of `MetricStore` to hold accumulated metrics, grouped by specific criteria. The criteria (`ResDefGroupingCriteria`) are needed in order to use the Batch Request.
        
    -   **Process Metrics from Channel**: Enters a loop to process metric definitions as they are sent through the `MetricDefinitionsChan` channel.
        
        -   **Update Metric Definitions**: Updates the `MetricDefinitions` if required. The metric definitions are only updated if they have expired. The `MetricDefinitions` are needed in the **Check Refresh Interval** step. In that way, if not expired in an upcoming fetch, the stored `MetricDefinitions` will be used, avoiding redundant API calls.
            
        -   **Group and Store Metrics**: Calls `GroupAndStoreMetrics` to group metrics and store them in `MetricStore`. Refer to the **GroupAndStoreMetrics Method**.
            
        -   **Process Stores**: If the store size reaches the batch API limit, it processes the store using the `processStore` function and collects metric values. That way, the Batch API will be used in the most efficient way. Refer to the **processStore and processAllStores Functions**.
            
    -   **Map and Publish Events**: Maps the collected metric values into events and publishes them using the `mapToEvents` method.
        
    -   **Error Handling**:
        
        -   **MetricDefinitionsChan is Closed**: In case `MetricDefinitionsChan` is closed, it processes all remaining metric stores using the `processAllStores` function and publishes the final set of events. The `MetricDefinitionsChan` can be closed in case all metric definitions have been collected by all goroutines. In that case, stores that have not reached the size of the batch API limit will be processed, collecting all the metric values.
            
        -   **Error received in ErrorChan**: Listens to `ErrorChan`. If an error happens during the **Check for Errors** step of metric definitions collection, we stop the data collection.
            
    -   **Terminate Loop**: Breaks the loop when both the data and error channels are closed.
        
    -   **Final Processing**: Processes all remaining metric stores using the `processAllStores` function and publishes the final set of events. This step is for safety reasons, in case the **MetricDefinitionsChan is Closed** step is not triggered. May be redundant.
        
5.  **GroupAndStoreMetrics Method**:
    
    -   **Group Metrics**: Groups metrics based on specific criteria which are Namespace, Subscription ID, Location, Names, aggregations, TimeGrain, and Dimensions. Batch API can be called for multiple resources only if those criteria are the same for all resources.
        
    -   **Check Update Requirement**: Checks if the metric needs to be collected again based on the time grain and the last collection time.
        
    -   **Store Metrics**: Adds the metrics to the appropriate `MetricStore`.
        
6.  **processStore and processAllStores Functions**:
    
    -   **Collect Metric Values**: Collects metric values for the metrics stored in the `MetricStore` using the batch API.
        
    -   **Clear Store**: Clears the metrics from the store after collecting the values. This is required so metric values for the same resources are not collected again in the same collection period.
        
    -   **Process All Stores**: Iterates over all metric stores and collects metric values for each, using the batch API.
        
7.  **GetMetricsInBatch Method**:
    
    -   **Prepare Batch Request**: Prepares a batch request for the metrics grouped by the same criteria.
        
    -   **Set Time Interval**: Sets the time interval for the metrics collection.
        
    -   **Add Filter Conditions**: Adds filter conditions for the metrics based on their dimensions.
        
    -   **Make API Call**: Makes a batch API call to Azure Monitor to retrieve the metric values.
        
        -   **Method Details**:
            
            -   **Client Creation**: Creates a new QueryResources client, setting up the endpoint, credentials, and options. For each different location, a new client is required.
                
            -   **Query Options**: Sets up the query options including time grain, filter, start time, end time, and top (limit).
                
            -   **Query Execution**: Calls the QueryResources method of the Azure Monitor service client, passing the resource IDs and query options.
                
            -   **Batch Processing**: Processes resource IDs in batches of BatchApiResourcesLimit (typically 50).
                
            -   **Handle Response**: Appends the metric data from the response to the result list.
                
    -   **Process Response**: Processes the API response, updates the metric registry, and appends the collected values to the metric definitions.