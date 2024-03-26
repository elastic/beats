// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

const resourceIDExtension = "/default"
const serviceTypeNamespaceExtension = "Services"

// mapMetrics should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetrics(client *azure.Client, resources []*armresources.GenericResourceExpanded, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {
	var metrics []azure.Metric
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
	for _, resource := range resources {
		for _, namespace := range namespaces {
			// resourceID will be different for a  serviceType namespace, format will be resourceID/service/default
			var resourceID = *resource.ID

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

					metrics = append(metrics, client.MapMetricByPrimaryAggregation(mets, *resource.ID, resourceID, namespace, dimensions, time)...)
				}
			}
		}
	}
	return metrics, nil
}

// groupOnTimeGrain - some metrics do not support the default timegrain value so the closest supported timegrain will be selected
func groupOnTimeGrain(list []armmonitor.MetricDefinition) map[string][]armmonitor.MetricDefinition {
	var groupedList = make(map[string][]armmonitor.MetricDefinition)

	for _, metric := range list {
		timegrain := retrieveSupportedMetricAvailability(metric.MetricAvailabilities)
		if _, ok := groupedList[timegrain]; !ok {
			groupedList[timegrain] = make([]armmonitor.MetricDefinition, 0)
		}
		groupedList[timegrain] = append(groupedList[timegrain], metric)
	}
	return groupedList
}

// retrieveSupportedMetricAvailability func will return the default timegrain if supported, else will return the next timegrain
func retrieveSupportedMetricAvailability(availabilities []*armmonitor.MetricAvailability) string {
	// common case in metrics supported by storage account - one availability
	if len(availabilities) == 1 {
		return *availabilities[0].TimeGrain
	}
	// check if the default timegrain is supported
	for _, availability := range availabilities {
		if *availability.TimeGrain == azure.DefaultTimeGrain {
			return azure.DefaultTimeGrain
		}
	}
	// select first timegrain, should be bigger than the min timegrain of 1M, timegrains are returned in asc order
	if *availabilities[0].TimeGrain != "PT1M" {
		return *availabilities[0].TimeGrain
	}
	return *availabilities[1].TimeGrain
}

// retrieveServiceNamespace func will check if the namespace is part of the service namespaces and returns the the selected name
func retrieveServiceNamespace(item string) string {
	for _, i := range storageServiceNamespaces {
		if defaultStorageAccountNamespace+i == item {
			return i
		}
	}
	return ""
}

// filterAllowedDimension func will filter out all unallowed dimensions
func filterAllowedDimension(metric armmonitor.MetricDefinition) []string {
	if metric.Dimensions == nil {
		return nil
	}
	var dimensions []string
	for _, dimension := range metric.Dimensions {
		for _, dim := range allowedDimensions {
			if dim == *dimension.Value {
				dimensions = append(dimensions, dim)
			}
		}
	}
	return dimensions
}

// groupMetricsByAllowedDimensions will group metrics by dimension names in order to reduce the number of api calls
func groupMetricsByAllowedDimensions(metrics []armmonitor.MetricDefinition) map[string][]armmonitor.MetricDefinition {
	var groupedMetrics = make(map[string][]armmonitor.MetricDefinition)
	for _, metric := range metrics {
		if dimensions := filterAllowedDimension(metric); len(dimensions) > 0 {
			for _, dimension := range dimensions {
				if _, ok := groupedMetrics[dimension]; !ok {
					groupedMetrics[dimension] = make([]armmonitor.MetricDefinition, 0)
				}
				groupedMetrics[dimension] = append(groupedMetrics[dimension], metric)
			}
		} else {
			if _, ok := groupedMetrics[azure.NoDimension]; !ok {
				groupedMetrics[azure.NoDimension] = make([]armmonitor.MetricDefinition, 0)
			}
			groupedMetrics[azure.NoDimension] = append(groupedMetrics[azure.NoDimension], metric)
		}
	}
	return groupedMetrics
}
