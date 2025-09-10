// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package monitor

import (
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// concurrentMapMetrics fetches concurrently metric definitions and writes them in MetricDefinitionsChan channel
func concurrentMapMetrics(client *azure.BatchClient, resources []*armresources.GenericResourceExpanded, resourceConfig azure.ResourceConfig, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		for _, resource := range resources {
			res, err := getMappedResourceDefinitions(client, *resource.ID, *resource.Location, client.Config.SubscriptionId, resourceConfig)
			if err != nil {
				client.ResourceConfigurations.ErrorChan <- err // Send error and stop processing
				return
			}
			client.ResourceConfigurations.MetricDefinitionsChan <- res
		}
	}()
}

// getMappedResourceDefinitions fetches metric definitions and maps the metric related configuration to relevant azure monitor api parameters
func getMappedResourceDefinitions(client *azure.BatchClient, resourceId string, location string, subscriptionId string, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {

	var metrics []azure.Metric
	// We use this map to avoid calling the metrics definition function for the same namespace and same resource
	// multiple times.
	namespaceMetrics := make(map[string]armmonitor.MetricDefinitionCollection)

	for _, metricConfig := range resourceConfig.Metrics {

		var err error

		metricDefinitions, exists := namespaceMetrics[metricConfig.Namespace]
		if !exists {
			metricDefinitions, err = client.AzureMonitorService.GetMetricDefinitionsWithRetry(resourceId, metricConfig.Namespace)
			if err != nil {
				return nil, err
			}
			namespaceMetrics[metricConfig.Namespace] = metricDefinitions
		}

		if len(metricDefinitions.Value) == 0 {
			if metricConfig.IgnoreUnsupported {
				client.Log.Infof(missingMetricDefinitions, resourceId, metricConfig.Namespace)
				continue
			}
			return nil, fmt.Errorf(missingMetricDefinitions, resourceId, metricConfig.Namespace)
		}

		// validate metric names and filter on the supported metrics
		supportedMetricNames, err := filterMetricNames(resourceId, metricConfig, metricDefinitions.Value)
		if err != nil {
			return nil, err
		}

		//validate aggregations and filter on supported aggregations
		metricGroups, err := validateAndGroupByConfiguredAggsAndTimegrain(
			supportedMetricNames, metricConfig, metricDefinitions.Value, client.Log)
		if err != nil {
			return nil, err
		}

		// map dimensions
		var dimensions []azure.Dimension
		if len(metricConfig.Dimensions) > 0 {
			for _, dimension := range metricConfig.Dimensions {
				dimensions = append(dimensions, azure.Dimension(dimension))
			}
		}
		for compositeKey, metricGroup := range metricGroups {
			var metricNames []string
			for _, metricName := range metricGroup {
				metricNames = append(metricNames, *metricName.Name.Value)
			}
			metrics = append(metrics, client.CreateMetric(resourceId, "", metricConfig.Namespace, location, subscriptionId, metricNames, compositeKey.aggregations, dimensions, compositeKey.timegrain))
		}
	}
	return metrics, nil
}
