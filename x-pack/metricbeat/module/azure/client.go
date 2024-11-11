// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

// MetricCollectionInfo contains information about the last time
// a metric was collected and the time grain used.
type MetricCollectionInfo struct {
	timestamp time.Time
	timeGrain string
}

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	mutex                  *sync.Mutex
	AzureMonitorService    Service
	Config                 Config
	ResourceConfigurations ResourceConfiguration
	Log                    *logp.Logger
	Resources              []Resource
	Resources2             map[string]*ResourceInfo
	//MetricsDefinitions     map[string][]Metric
	Resources2Updated                time.Time
	EndOfMetricsDefinitionThrottling time.Time
	MetricRegistry                   *MetricRegistry
}

// mapResourceMetrics function type will map the configuration options to client metrics (depending on the metricset)
// type mapResourceMetrics func(client *Client, resources []*armresources.GenericResourceExpanded, resourceConfig ResourceConfig) ([]Metric, error)
type mapResourceMetrics func(client *Client, resources *[]Resource, resourceConfig ResourceConfig) ([]Metric, error)

// NewClient instantiates the Azure monitoring client
func NewClient(config Config) (*Client, error) {
	azureMonitorService, err := NewService(config)
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("azure monitor client")

	client := &Client{
		mutex:               &sync.Mutex{},
		AzureMonitorService: azureMonitorService,
		Config:              config,
		Log:                 logger,
		MetricRegistry:      NewMetricRegistry(logger),
	}

	client.Resources2 = map[string]*ResourceInfo{}
	//client.MetricsDefinitions = map[string][]Metric{}

	client.ResourceConfigurations.RefreshInterval = config.RefreshListInterval

	return client, nil
}

func (client *Client) UpdateMetricsDefinitions(resourceID string, metrics []Metric) {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	client.Resources2[resourceID].definitions = metrics
	client.Resources2[resourceID].definitionsUpdated = time.Now()
}

func (client *Client) RefreshResources() error {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	if time.Since(client.Resources2Updated).Seconds() < client.Config.RefreshListInterval.Seconds() {
		return nil
	}

	if len(client.Config.Resources) == 0 {
		return fmt.Errorf("no resource options defined")
	}

	//// check if refresh interval has been set and if it has expired
	//if !client.ResourceConfigurations.Expired() {
	//	return nil
	//}

	existingResources := map[string]struct{}{}
	for id := range client.Resources2 {
		existingResources[id] = struct{}{}
	}

	//reset client resources
	//client.Resources = []Resource{}
	for _, resource := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.Id, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = fmt.Errorf("failed to retrieve resources: %w", err)
			return err
		}

		if len(resourceList) == 0 {
			err = fmt.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.Id, resource.Group, resource.Type, resource.Query)
			client.Log.Error(err)
			continue
		}

		// Map resources to the client
		for _, resource := range resourceList {
			if _, ok := client.Resources2[*resource.ID]; !ok {
				client.Resources2[*resource.ID] = &ResourceInfo{resource: Resource{
					Id:             *resource.ID,
					Name:           *resource.Name,
					Location:       *resource.Location,
					Type:           *resource.Type,
					Group:          getResourceGroupFromId(*resource.ID),
					Tags:           mapTags(resource.Tags),
					SubscriptionID: client.Config.SubscriptionId,
				}}
			}

			delete(existingResources, *resource.ID)
			//if !containsResource(*resource.ID, client.Resources) {
			//	client.Resources = append(client.Resources,
			//}
		}

		//// Collects and stores metrics definitions for the cloud resources.
		//resourceMetrics, err := fn(client, resourceList, resource)
		//if err != nil {
		//	return err
		//}

		//metrics = append(metrics, resourceMetrics...)
	}

	if len(existingResources) > 0 {
		for idToDelete := range existingResources {
			delete(client.Resources2, idToDelete)
		}
	}

	client.Resources2Updated = time.Now()

	return nil
}

func (client *Client) RefreshResourceMetricsDefinitions(resourceID string, fn mapResourceMetrics) error {
	if time.Since(client.Resources2[resourceID].definitionsUpdated).Seconds() < client.Config.RefreshListInterval.Seconds() {
		return nil
	}

	if time.Now().Before(client.EndOfMetricsDefinitionThrottling) {
		// Azure is throttling API requests to get metrics definitions
		return nil
	}

	var metricsDefinitions []Metric

	//resource, ok := client.Resources2[resourceID]
	//if !ok {
	//	return fmt.Errorf("resource %s not found", resourceID)
	//}

	resources := []Resource{client.Resources2[resourceID].resource}

	for _, resourceConfig := range client.Config.Resources {
		metricsDefinition, err := fn(client, &resources, resourceConfig)
		if err != nil {
			var throttlingError *ThrottlingError
			if errors.As(err, throttlingError) {
				client.EndOfMetricsDefinitionThrottling = throttlingError.End
			}

			return err
		}

		metricsDefinitions = append(metricsDefinitions, metricsDefinition...)
	}

	//client.Resources2[resourceID].definitions = metricsDefinitions
	//client.Resources2[resourceID].definitionsUpdated = time.Now()
	client.UpdateMetricsDefinitions(resourceID, metricsDefinitions)

	return nil
}

//func (client *Client) UpdateResourceMetricsDefinitions(resourceID string, metrics []Metric) {
//	client.Resources2[resourceID].definitions = metrics
//	client.Resources2[resourceID].definitionsUpdated = time.Now()
//}

func (client *Client) GetMetricsDefinitions() []Metric {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	var metrics []Metric

	for _, resource := range client.Resources2 {
		if len(resource.definitions) > 0 {
			metrics = append(metrics, resource.definitions...)
		}
	}

	return metrics
}

// InitResources function will retrieve and validate the resources configured by the users and then map the information configured to client metrics.
// the mapMetric function sent in this case will handle the mapping part as different metric and aggregation options work for different metricsets
func (client *Client) InitResources(fn mapResourceMetrics) error {
	if len(client.Config.Resources) == 0 {
		return fmt.Errorf("no resource options defined")
	}

	// check if refresh interval has been set and if it has expired
	if !client.ResourceConfigurations.Expired() {
		return nil
	}

	var metrics []Metric
	//reset client resources
	client.Resources = []Resource{}
	for _, resource := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.Id, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = fmt.Errorf("failed to retrieve resources: %w", err)
			return err
		}

		if len(resourceList) == 0 {
			err = fmt.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.Id, resource.Group, resource.Type, resource.Query)
			client.Log.Error(err)
			continue
		}

		// Map resources to the client
		for _, resource := range resourceList {
			if !containsResource(*resource.ID, client.Resources) {
				client.Resources = append(client.Resources, Resource{
					Id:             *resource.ID,
					Name:           *resource.Name,
					Location:       *resource.Location,
					Type:           *resource.Type,
					Group:          getResourceGroupFromId(*resource.ID),
					Tags:           mapTags(resource.Tags),
					SubscriptionID: client.Config.SubscriptionId})
			}
		}

		// Collects and stores metrics definitions for the cloud resources.
		resourceMetrics, err := fn(client, &client.Resources, resource)
		if err != nil {
			return err
		}

		metrics = append(metrics, resourceMetrics...)
	}

	// users could add or remove resources while metricbeat is running so we could encounter the situation where
	// resources are unavailable we log an error message (see above) we also log a debug message when absolutely no
	// resources are found
	if len(metrics) == 0 {
		client.Log.Debug("no resources were found based on all the configurations options entered")
	}
	client.ResourceConfigurations.Metrics = metrics

	return nil
}

// buildTimespan returns the timespan for the metric values given the reference time,
// time grain and collection period.
//
// (1) When the collection period is greater than the time grain, the timespan
// will be:
//
// |                                            time grain
// │                                          │◀──(PT1M)──▶ │
// │                                                        │
// ├──────────────────────────────────────────┼─────────────┼─────────────
// │                                                        │
// │                       timespan           │             │
// |◀───────────────────────(5min)─────────────────────────▶│
// │                                          │             │
// |                        period                          │
// │◀───────────────────────(5min)────────────┼────────────▶│
// │                                                        │
// │                                          │             │
// |                                                        │
// |                                                       Now
// |                                                        │
//
// In this case, the API will return five metric values, because
// the time grain is 1 minute and the timespan is 5 minutes.
//
// (2) When the collection period is equal to the time grain,
// the timespan will be:
//
// |
// │                       time grain                       │
// |◀───────────────────────(5min)─────────────────────────▶│
// │                                                        │
// ├────────────────────────────────────────────────────────┼─────────────
// │                                                        │
// │                       timespan                         │
// |◀───────────────────────(5min)─────────────────────────▶│
// │                                                        │
// |                        period                          │
// │◀───────────────────────(5min)─────────────────────────▶│
// │                                                        │
// │                                                        │
// |                                                        │
// |                                                       Now
// |                                                        │
//
// In this case, the API will return one metric value.
//
// (3) When the collection period is less than the time grain,
// the timespan will be:
//
// |                                              period
// │                                          │◀──(5min)──▶ │
// │                                                        │
// ├──────────────────────────────────────────┼─────────────┼─────────────
// │                                                        │
// │                       timespan           │             │
// |◀───────────────────────(60min)────────────────────────▶│
// │                                          │             │
// |                      time grain                        │
// │◀───────────────────────(PT1H)────────────┼────────────▶│
// │                                                        │
// │                                          │             │
// |                                                       Now
// |                                                        │
// |
//
// In this case, the API will return one metric value.
func buildTimespan(referenceTime time.Time, timeGrain string, collectionPeriod time.Duration) string {
	timespanDuration := max(asDuration(timeGrain), collectionPeriod)

	endTime := referenceTime
	startTime := endTime.Add(timespanDuration * -1)

	return fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
}

func (client *Client) GetMetricValues2(referenceTime time.Time, metrics []Metric, reporter mb.ReporterV2) []Metric {
	var result []Metric

	// Same end time for all metrics in the same batch.
	interval := client.Config.Period

	// Fetch in the range [{-2 x INTERVAL},{-1 x INTERVAL}) with a delay of {INTERVAL}.
	endTime := referenceTime.Add(interval * (-1))
	startTime := endTime.Add(interval * (-1))

	index := map[string][]Metric{}

	// Group metrics by the following keys:
	for _, metric := range metrics {
		var dimensions []string
		for _, d := range metric.Dimensions {
			dimensions = append(dimensions, d.Name)
		}

		key := fmt.Sprintf(
			"%s-%s-%s-%s-%s-%s-%s",
			metric.Namespace,
			metric.SubscriptionID,
			metric.Location,
			strings.Join(metric.Names, ","),
			metric.Aggregations,
			strings.Join(dimensions, ","),
			metric.TimeGrain,
		)
		if _, ok := index[key]; !ok {
			index[key] = []Metric{
				metric,
			}
		} else {
			index[key] = append(index[key], metric)
		}
	}

	for _, metricsDefinitions := range index {
		//uniqueResourceIDs := map[string]bool{}
		//for _, metric := range metricsDefinitions {
		//	uniqueResourceIDs[metric.ResourceId] = true
		//}

		var resourceIDs []*string
		for _, m := range metricsDefinitions {
			if !client.MetricRegistry.NeedsUpdate(referenceTime, m) {
				continue
			}
			var resourceID = m.ResourceId
			resourceIDs = append(resourceIDs, &resourceID)
		}

		// build the 'filter' parameter which will contain any dimensions configured
		var filter string
		if len(metricsDefinitions[0].Dimensions) > 0 {
			var filterList []string
			for _, dim := range metricsDefinitions[0].Dimensions {
				filterList = append(filterList, dim.Name+" eq '"+dim.Value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}

		r, err := client.AzureMonitorService.QueryResources(
			//fmt.Println(
			resourceIDs,
			metricsDefinitions[0].SubscriptionID,
			metricsDefinitions[0].Namespace,
			metricsDefinitions[0].TimeGrain,
			startTime.Format("2006-01-02T15:04:05.000Z07:00"),
			endTime.Format("2006-01-02T15:04:05.000Z07:00"),
			//startTime.Format(time.RFC3339),
			//endTime.Format(time.RFC3339),
			metricsDefinitions[0].Names,
			metricsDefinitions[0].Aggregations,
			filter, // dimensions
		)
		if err != nil {
			err = fmt.Errorf("error while listing metric values by resource ID %s and namespace  %s: %w", metricsDefinitions[0].ResourceSubId, metricsDefinitions[0].Namespace, err)
			client.Log.Error(err)
			reporter.Error(err)
			continue
		}

		if len(r) != len(metricsDefinitions) {
			err = fmt.Errorf("error while listing metric values by resource ID %s and namespace  %s: expected %d values, got %d", metricsDefinitions[0].ResourceSubId, metricsDefinitions[0].Namespace, len(metricsDefinitions), len(r))
			client.Log.Error(err)
			reporter.Error(err)
			continue
		}

		for i, _ := range r {
			client.MetricRegistry.Update(metricsDefinitions[i], MetricCollectionInfo{
				timeGrain: *r[i].Interval,
				timestamp: referenceTime,
			})
			metricsDefinitions[i].Values = append(metricsDefinitions[i].Values, mapMetricValues2(r[i])...)
		}

		result = append(result, metricsDefinitions...)
	}

	return result
}

// GetMetricValues returns the metric values for the given cloud resources.
func (client *Client) GetMetricValues(referenceTime time.Time, metrics []Metric, reporter mb.ReporterV2) []Metric {
	var result []Metric

	for _, metric := range metrics {
		timespan := buildTimespan(referenceTime, metric.TimeGrain, client.Config.Period)

		//
		// Before fetching the metric values, check if the metric
		// has been collected within the time grain.
		//
		// Why do we need this?
		//
		// Some metricsets contains metrics with long time grains (e.g. 1 hour).
		//
		// If we collect the metric values every 5 minutes, we will end up fetching
		// the same data over and over again for all metrics with a time grain
		// larger than 5 minutes.
		//
		// The registry keeps track of the last timestamp the metricset collected
		// the metric values and the time grain used.
		//
		// By comparing the last collection time with the current time, and
		// the time grain of the metric, we can determine if the metric needs
		// to be collected again, or if we can skip it.
		//
		if !client.MetricRegistry.NeedsUpdate(referenceTime, metric) {
			continue
		}

		// build the 'filter' parameter which will contain any dimensions configured
		var filter string
		if len(metric.Dimensions) > 0 {
			var filterList []string
			for _, dim := range metric.Dimensions {
				filterList = append(filterList, dim.Name+" eq '"+dim.Value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}

		// Fetch the metric values from the Azure API.
		resp, timeGrain, err := client.AzureMonitorService.GetMetricValues(
			metric.ResourceSubId,
			metric.Namespace,
			metric.TimeGrain,
			timespan,
			metric.Names,
			metric.Aggregations,
			filter,
		)
		if err != nil {
			err = fmt.Errorf("error while listing metric values by resource ID %s and namespace  %s: %w", metric.ResourceSubId, metric.Namespace, err)
			client.Log.Error(err)
			reporter.Error(err)

			// Skip this metric and continue with the next one.
			break
		}

		// Update the metric registry with the latest timestamp and
		// time grain for each metric.
		//
		// We track the time grain Azure used for this metric values from
		// the API response.
		client.MetricRegistry.Update(metric, MetricCollectionInfo{
			timeGrain: timeGrain,
			timestamp: referenceTime,
		})

		//for i, currentMetric := range client.ResourceConfigurations.Metrics {
		//	if matchMetrics(currentMetric, metric) {
		//		// Map the metric values from the API response.
		//		current := mapMetricValues(resp, currentMetric.Values)
		//		client.ResourceConfigurations.Metrics[i].Values = current
		//
		//		// Some predefined metricsets configuration do not have a time grain.
		//		// Here is an example:
		//		// https://github.com/elastic/beats/blob/024a9cec6608c6f371ad1cb769649e024124ff92/x-pack/metricbeat/module/azure/database_account/manifest.yml#L11-L13
		//		//
		//		// Predefined metricsets sometimes have long lists of metrics
		//		// with no time grains. Or users can configure their own
		//		// custom metricsets with no time grain.
		//		//
		//		// In this case, we track the time grain returned by the API. Azure
		//		// provides a default time grain for each metric.
		//		if client.ResourceConfigurations.Metrics[i].TimeGrain == "" {
		//			client.ResourceConfigurations.Metrics[i].TimeGrain = timeGrain
		//		}
		//
		//		result = append(result, client.ResourceConfigurations.Metrics[i])
		//	}
		//}

		metric.Values = mapMetricValues(resp)

		//(*metrics)[i].Values = mapMetricValues(resp)
		//result = append(result, mapMetricValues(resp)...)
		result = append(result, metric)
	}

	return result
}

// CreateMetricDefinition function will create a client metric based on the resource and metrics configured
func (client *Client) CreateMetricDefinition(
	resourceId string,
	subResourceId string,
	subscriptionID string,
	location string,
	namespace string,
	metrics []string,
	aggregations string,
	dimensions []Dimension,
	timeGrain string,
) Metric {
	if subResourceId == "" {
		subResourceId = resourceId
	}
	met := Metric{
		ResourceId:     resourceId,
		ResourceSubId:  subResourceId,
		SubscriptionID: subscriptionID,
		Location:       location,
		Namespace:      namespace,
		Names:          metrics,
		Dimensions:     dimensions,
		Aggregations:   aggregations,
		TimeGrain:      timeGrain,
	}

	for _, prevMet := range client.ResourceConfigurations.Metrics {
		if len(prevMet.Values) != 0 && matchMetrics(prevMet, met) {
			met.Values = prevMet.Values
		}
	}

	return met
}

// MapMetricByPrimaryAggregation will map the primary aggregation of the metric definition to the client metric
func (client *Client) MapMetricByPrimaryAggregation(metrics []azquery.MetricDefinition, resourceId, subscriptionID, location, subResourceId, namespace string, dim []Dimension, timeGrain string) []Metric {
	clientMetrics := make([]Metric, 0)
	metricGroups := make(map[string][]azquery.MetricDefinition)

	for _, met := range metrics {
		metricGroups[string(*met.PrimaryAggregationType)] = append(metricGroups[string(*met.PrimaryAggregationType)], met)
	}

	for key, metricGroup := range metricGroups {
		var metricNames []string
		for _, metricName := range metricGroup {
			metricNames = append(metricNames, *metricName.Name.Value)
		}
		clientMetrics = append(clientMetrics, client.CreateMetricDefinition(
			resourceId,
			subResourceId,
			subscriptionID,
			location,
			namespace,
			metricNames,
			key,
			dim,
			timeGrain,
		))
	}

	return clientMetrics
}

// GetVMForMetadata func will retrieve the VM details in order to fill in the cloud metadata
// and also update the client resources
func (client *Client) GetVMForMetadata(resource *Resource, referencePoint KeyValuePoint) VmResource {
	var (
		vm           VmResource
		resourceName = resource.Name
		resourceId   = resource.Id
	)

	// Search the dimensions for the "VMName" dimension. This dimension is present for VM Scale Sets.
	if dimensionValue, ok := getDimension("VMName", referencePoint.Dimensions); ok {
		instanceId := getInstanceId(dimensionValue)
		if instanceId != "" {
			resourceId += fmt.Sprintf("/virtualMachines/%s", instanceId)
			resourceName = dimensionValue
		}
	}

	// if vm has been already added to the resource then it should be returned
	if existingVM, ok := getVM(resourceName, resource.Vms); ok {
		return existingVM
	}

	// an additional call is necessary in order to retrieve the vm specific details
	expandedResource, err := client.AzureMonitorService.GetResourceDefinitionById(resourceId)
	if err != nil {
		client.Log.Error(err, "could not retrieve the resource details by resource ID %s", resourceId)
		return VmResource{}
	}

	vm.Name = *expandedResource.Name

	if expandedResource.Properties != nil {
		if properties, ok := expandedResource.Properties.(map[string]interface{}); ok {
			if hardware, ok := properties["hardwareProfile"]; ok {
				if vmSz, ok := hardware.(map[string]interface{})["vmSize"]; ok {
					vm.Size = vmSz.(string)
				}
				if vmID, ok := properties["vmId"]; ok {
					vm.Id = vmID.(string)
				}
			}
		}
	}

	if len(vm.Size) == 0 && expandedResource.SKU != nil && expandedResource.SKU.Name != nil {
		vm.Size = *expandedResource.SKU.Name
	}

	// the client resource and selected resources are being updated in order to avoid additional calls
	client.AddVmToResource(resource.Id, vm)

	resource.Vms = append(resource.Vms, vm)

	return vm
}

// GetResourceForMetaData will retrieve resource details for the selected metric configuration
func (client *Client) GetResourceForMetaData(grouped Metric) Resource {
	for _, res := range client.Resources {
		if res.Id == grouped.ResourceId {
			return res
		}
	}
	return Resource{}
}

func (client *Client) LookupResource(resourceId string) Resource {
	client.mutex.Lock()
	defer client.mutex.Unlock()
	//for _, res := range client.Resources {
	//	if res.Id == resourceId {
	//		return res
	//	}
	//}
	if res, ok := client.Resources2[resourceId]; ok {
		return res.resource
	}

	return Resource{}
}

// AddVmToResource will add the vm details to the resource
func (client *Client) AddVmToResource(resourceId string, vm VmResource) {
	if len(vm.Id) > 0 && len(vm.Name) > 0 {
		for i, res := range client.Resources {
			if res.Id == resourceId {
				client.Resources[i].Vms = append(client.Resources[i].Vms, vm)
			}
		}
	}
}
