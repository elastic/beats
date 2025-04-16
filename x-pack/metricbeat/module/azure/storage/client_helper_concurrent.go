// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// concurrentMapMetrics fetches concurrently metric definitions and writes them in MetricDefinitionsChan channel
func concurrentMapMetrics(client *azure.BatchClient, resources []*armresources.GenericResourceExpanded, resourceConfig azure.ResourceConfig, wg *sync.WaitGroup) {
	// list all storage account namespaces for this metricset
	namespaces := []string{defaultStorageAccountNamespace}
	// if serviceType is configured, add only the selected serviceType namespaces
	if len(resourceConfig.ServiceType) > 0 {
		for _, selectedServiceNamespace := range resourceConfig.ServiceType {
			namespaces = append(namespaces, fmt.Sprintf("%s/%s%s", defaultStorageAccountNamespace, selectedServiceNamespace, serviceTypeNamespaceExtension))
		}
	} else {
		for _, service := range storageServiceNamespaces {
			namespaces = append(namespaces, fmt.Sprintf("%s%s", defaultStorageAccountNamespace, service))
		}
	}
	go func() {
		defer wg.Done()
		for _, resource := range resources {
			res, err := getStorageMappedResourceDefinitions(client, *resource.ID, *resource.Location, client.Config.SubscriptionId, namespaces)
			if err != nil {
				client.ResourceConfigurations.ErrorChan <- err // Send error and stop processing
				return
			}
			client.ResourceConfigurations.MetricDefinitionsChan <- res
		}
	}()
}

// getStorageMappedResourceDefinitions fetches metric definitions and maps the metric related configuration to relevant azure monitor api parameters
func getStorageMappedResourceDefinitions(client *azure.BatchClient, resourceId string, location string, subscriptionId string, namespaces []string) ([]azure.Metric, error) {

	var metrics []azure.Metric

	for _, namespace := range namespaces {
		// resourceID will be different for a  serviceType namespace, format will be resourceID/service/default
		var resourceID = resourceId
		if i := retrieveServiceNamespace(namespace); i != "" {
			resourceID += i + resourceIDExtension
		}
		// get all metric definitions supported by the namespace provided
		metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitionsWithRetry(resourceID, namespace)
		if err != nil {
			return nil, err
		}
		if len(metricDefinitions.Value) == 0 {
			return nil, fmt.Errorf("no metric definitions were found for resource %s and namespace %s", resourceID, namespace)
		}
		var filteredMetricDefinitions []armmonitor.MetricDefinition
		for _, metricDefinition := range metricDefinitions.Value {
			filteredMetricDefinitions = append(filteredMetricDefinitions, *metricDefinition)
		}
		// some metrics do not support the default PT5M timegrain so they will have to be grouped in a different API call, else call will fail
		groupedMetrics := groupOnTimeGrain(filteredMetricDefinitions)
		for time, groupedMetricList := range groupedMetrics {
			// metrics will have to be grouped by allowed dimensions
			dimMetrics := groupMetricsByAllowedDimensions(groupedMetricList)
			for dimension, mets := range dimMetrics {
				var dimensions []azure.Dimension
				if dimension != azure.NoDimension {
					dimensions = []azure.Dimension{{Name: dimension, Value: "*"}}
				}

				metrics = append(metrics, client.MapMetricByPrimaryAggregation(mets, resourceId, location, subscriptionId, resourceID, namespace, dimensions, time)...)
			}
		}
	}
	return metrics, nil
}
