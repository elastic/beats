// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
)

// BatchClient represents the azure batch client which will make use of the azure sdk go metrics related clients
type BatchClient struct {
	*BaseClient
	ResourceConfigurations ConcurrentResourceConfig
}

// Resource definitions grouping criteria
type ResDefGroupingCriteria struct {
	Namespace      string
	SubscriptionID string
	Location       string
	Names          string
	Aggregations   string
	TimeGrain      string
	Dimensions     string
}

// concurrentMapResourceMetrics function type will map the configuration options to Batch Client metrics (depending on the metricset)
type concurrentMapResourceMetrics func(client *BatchClient, resources []*armresources.GenericResourceExpanded, resourceConfig ResourceConfig, wg *sync.WaitGroup)

// NewBatchClient instantiates the Azure monitoring batch client
func NewBatchClient(config Config) (*BatchClient, error) {
	azureMonitorService, err := NewService(config)
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("azure monitor client")

	client := &BatchClient{
		BaseClient: &BaseClient{
			AzureMonitorService: azureMonitorService,
			Config:              config,
			Log:                 logger,
			MetricRegistry:      NewMetricRegistry(logger),
		},
	}
	client.ResourceConfigurations.MetricDefinitions = MetricDefinitions{
		Update:  true,
		Metrics: make(map[string][]Metric),
	}
	client.ResourceConfigurations.RefreshInterval = config.RefreshListInterval
	client.ResourceConfigurations.MetricDefinitionsChan = nil
	client.ResourceConfigurations.ErrorChan = nil

	return client, nil
}

// InitResources function will retrieve and validate the resources configured by the users and then map the information configured to client metrics.
// the mapMetric function sent in this case will handle the mapping part as different metric and aggregation options work for different metricsets
func (client *BatchClient) InitResources(fn concurrentMapResourceMetrics) error {
	if len(client.Config.Resources) == 0 {
		return fmt.Errorf("no resource options defined")
	}

	// check if refresh interval has been set and if it has expired
	if !client.ResourceConfigurations.Expired() {
		client.Log.Debug("MetricDefinitions are not expired. Writing metrics to MetricDefinitionsChan")
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		// MetricDefinitions do not need update
		client.ResourceConfigurations.MetricDefinitions.Update = false
		go func() {
			defer close(client.ResourceConfigurations.MetricDefinitionsChan)
			defer close(client.ResourceConfigurations.ErrorChan)
			for _, metrics := range client.ResourceConfigurations.MetricDefinitions.Metrics {
				client.ResourceConfigurations.MetricDefinitionsChan <- metrics
			}
		}()
		return nil
	}
	// MetricDefinitions need update
	client.ResourceConfigurations.MetricDefinitions.Update = true

	// Initialize a WaitGroup to track all goroutines
	var wg sync.WaitGroup
	//reset client resources
	client.Resources = []Resource{}
	for _, resourceConfig := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resourceConfig.Id, resourceConfig.Group, resourceConfig.Type, resourceConfig.Query)
		if err != nil {
			err = fmt.Errorf("failed to retrieve resources: %w", err)
			return err
		}

		if len(resourceList) == 0 {
			err = fmt.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resourceConfig.Id, resourceConfig.Group, resourceConfig.Type, resourceConfig.Query)
			client.Log.Error(err)
			continue
		}
		// create the channels if they are not already created by a previous itteration
		if client.ResourceConfigurations.MetricDefinitionsChan == nil && client.ResourceConfigurations.ErrorChan == nil {
			client.ResourceConfigurations.MetricDefinitionsChan = make(chan []Metric)
			client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		}

		// Map resources to the client
		for _, resource := range resourceList {
			if !containsResource(*resource.ID, client.Resources) {
				client.Resources = append(client.Resources, Resource{
					Id:           *resource.ID,
					Name:         *resource.Name,
					Location:     *resource.Location,
					Type:         *resource.Type,
					Group:        getResourceGroupFromId(*resource.ID),
					Tags:         mapTags(resource.Tags),
					Subscription: client.Config.SubscriptionId})
			}
		}

		// Collects and stores metrics definitions for the cloud resources.
		wg.Add(1)
		fn(client, resourceList, resourceConfig, &wg)
		client.Log.Infof("Finished collection with %d metric definitions", len(resourceList))
	}
	go func() {
		wg.Wait() // Wait for all the resource collection goroutines to finish
		// Once all the goroutines are done, close the channels
		client.Log.Debug("All collections finished. Closing channels ")
		close(client.ResourceConfigurations.MetricDefinitionsChan)
		close(client.ResourceConfigurations.ErrorChan)
	}()
	return nil
}

// GetMetricsInBatch will query the batch API for each group
func (client *BatchClient) GetMetricsInBatch(groupedMetrics map[ResDefGroupingCriteria][]Metric, referenceTime time.Time, reporter mb.ReporterV2) []Metric {
	var result []Metric
	for criteria, metricsDefinitions := range groupedMetrics {
		// Same end time for all metrics in the same batch.
		interval := client.Config.Period

		// // Fetch in the range [{-2 x INTERVAL},{-1 x INTERVAL}) with a delay of {INTERVAL}.
		endTime := referenceTime
		timespanDuration := max(asDuration(criteria.TimeGrain), interval)
		startTime := endTime.Add(timespanDuration * -1)
		// Limit batch size to 50 resources (if you have more, you can split the batch)
		filter := ""
		if len(metricsDefinitions[0].Dimensions) > 0 {
			var filterList []string
			for _, dim := range metricsDefinitions[0].Dimensions {
				filterList = append(filterList, dim.Name+" eq '"+dim.Value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}
		for i := 0; i < len(metricsDefinitions); i += BatchApiResourcesLimit {
			end := i + BatchApiResourcesLimit
			if end > len(metricsDefinitions) {
				end = len(metricsDefinitions)
			}

			// Slice the metrics to form the batch request
			batchMetrics := metricsDefinitions[i:end]

			// Slice the Metric Names by batches of 20 due to batch api limitation
			names := strings.Split(criteria.Names, ",")
			for j := 0; j < len(names); j += metricNameLimit {
				endMetric := j + metricNameLimit
				if endMetric > len(names) {
					endMetric = len(names)
				}

				// Make the batch API call (adjust parameters as needed)
				response, err := client.AzureMonitorService.QueryResources(
					getResourceIDs(batchMetrics), // Get the resource IDs from the batch
					criteria.SubscriptionID,
					criteria.Namespace,
					criteria.TimeGrain,
					startTime.Format("2006-01-02T15:04:05.000Z07:00"),
					endTime.Format("2006-01-02T15:04:05.000Z07:00"),
					names[j:endMetric],
					strings.ToLower(batchMetrics[0].Aggregations),
					filter,
					criteria.Location,
				)
				if err != nil {
					err = fmt.Errorf("error while listing metric values by resource ID %s and namespace  %s: %w", metricsDefinitions[0].ResourceSubId, metricsDefinitions[0].Namespace, err)
					client.Log.Error(err)
					reporter.Error(err)
					continue
				}

				// Process the response as needed
				for i, v := range response {
					client.MetricRegistry.Update(metricsDefinitions[i], MetricCollectionInfo{
						timeGrain: *response[i].Interval,
						timestamp: referenceTime,
					})
					values := mapBatchMetricValues(client, v)
					metricsDefinitions[i].Values = append(metricsDefinitions[i].Values, values...)
					if metricsDefinitions[i].TimeGrain == "" {
						metricsDefinitions[i].TimeGrain = *response[i].Interval
					}

				}

				result = append(result, metricsDefinitions...)

			}
		}
	}

	return result
}

// GroupAndStoreMetrics groups received metricsDefinitions and stores them in a in memory store
func (client *BatchClient) GroupAndStoreMetrics(metricsDefinitions []Metric, referenceTime time.Time, store map[ResDefGroupingCriteria]*MetricStore) {
	for _, metric := range metricsDefinitions {

		criteria := ResDefGroupingCriteria{
			Namespace:      metric.Namespace,
			SubscriptionID: metric.SubscriptionId,
			Location:       metric.Location,
			Names:          strings.Join(metric.Names, ","),
			TimeGrain:      metric.TimeGrain,
			Dimensions:     getDimensionKey(metric.Dimensions),
		}

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
		if _, exists := store[criteria]; !exists {
			store[criteria] = &MetricStore{}
		}
		store[criteria].AddMetric(metric)
	}
}

// CreateMetric function will create a client metric based on the resource and metrics configured
func (client *BatchClient) CreateMetric(resourceId string, subResourceId string, namespace string, location string, subscriptionId string, metrics []string, aggregations string, dimensions []Dimension, timeGrain string) Metric {
	if subResourceId == "" {
		subResourceId = resourceId
	}
	met := Metric{
		ResourceId:     resourceId,
		ResourceSubId:  subResourceId,
		Namespace:      namespace,
		Names:          metrics,
		Dimensions:     dimensions,
		Aggregations:   aggregations,
		TimeGrain:      timeGrain,
		Location:       location,
		SubscriptionId: subscriptionId,
	}
	if prevMetrics, ok := client.ResourceConfigurations.MetricDefinitions.Metrics[resourceId]; ok {
		for _, prevMet := range prevMetrics {
			if len(prevMet.Values) != 0 && matchMetrics(prevMet, met) {
				met.Values = prevMet.Values
			}
		}
	}

	return met
}

// MapMetricByPrimaryAggregation will map the primary aggregation of the metric definition to the client metric
func (client *BatchClient) MapMetricByPrimaryAggregation(metrics []armmonitor.MetricDefinition, resourceId string, location string, subscriptionId string, subResourceId string, namespace string, dim []Dimension, timeGrain string) []Metric {
	clientMetrics := make([]Metric, 0)
	metricGroups := make(map[string][]armmonitor.MetricDefinition)

	for _, met := range metrics {
		metricGroups[string(*met.PrimaryAggregationType)] = append(metricGroups[string(*met.PrimaryAggregationType)], met)
	}

	for key, metricGroup := range metricGroups {
		var metricNames []string
		for _, metricName := range metricGroup {
			metricNames = append(metricNames, *metricName.Name.Value)
		}
		clientMetrics = append(clientMetrics, client.CreateMetric(resourceId, subResourceId, namespace, location, subscriptionId, metricNames, key, dim, timeGrain))
	}

	return clientMetrics
}

// NewMockBatchClient instantiates a new batch client with the mock azure service
func NewMockBatchClient() *BatchClient {
	azureMockService := new(MockService)
	logger := logp.NewLogger("test azure monitor")
	client := &BatchClient{
		BaseClient: &BaseClient{
			AzureMonitorService: azureMockService,
			Config:              Config{},
			Log:                 logger,
			MetricRegistry:      NewMetricRegistry(logger),
		},
	}
	return client
}
