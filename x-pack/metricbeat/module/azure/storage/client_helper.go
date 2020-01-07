// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

const resourceIDExtension = "/default"
const serviceTypeNamespaceExtension = "Services"

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetric(client *azure.Client, metric azure.MetricConfig, resource resources.GenericResource) ([]azure.Metric, error) {
	var metrics []azure.Metric
	//check if no metric names are configured
	if metric.Name == nil {
		return nil, nil
	}
	var namespaces []insights.MetricNamespace
	// return all namespaces supported for this resource
	response, err := client.AzureMonitorService.GetMetricNamespaces(*resource.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "no metric namespaces were found for resource %s", *resource.ID)
	}
	namespaces = append(namespaces, *response.Value...)

	// return all service namespaces for this resource (format of resource id will be resource path/servicetype/default) if serviceType is not configured
	var serviceNamespaces []string
	if len(metric.CustomFields.ServiceType) > 0 {
		for _, ser := range metric.CustomFields.ServiceType {
			serviceNamespaces = append(serviceNamespaces, ser+serviceTypeNamespaceExtension)
		}
	} else {
		serviceNamespaces = storageServiceNamespaces
	}
	for _, serviceNamespace := range serviceNamespaces {
		response, err = client.AzureMonitorService.GetMetricNamespaces(fmt.Sprintf("%s/%s%s", *resource.ID, serviceNamespace, resourceIDExtension))
		if err != nil {
			return nil, errors.Wrapf(err, "no metric namespaces were found for resource %s", *resource.ID)
		}
		namespaces = append(namespaces, *response.Value...)
	}

	for _, namespace := range namespaces {
		var resourceID = *resource.ID
		// get all metric definitions supported by the namespace provided
		if i := retrieveServiceNamespace(*namespace.Properties.MetricNamespaceName); i != "" {
			resourceID += i + resourceIDExtension
		}
		metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(resourceID, *namespace.Properties.MetricNamespaceName)
		if err != nil {
			return nil, errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s.", resourceID, *namespace.Properties.MetricNamespaceName)
		}
		if len(*metricDefinitions.Value) == 0 {
			return nil, errors.Errorf("no metric definitions were found for resource %s and namespace %s.", resourceID, *namespace.Properties.MetricNamespaceName)
		}
		var filteredMetricDefinitions []insights.MetricDefinition
		for _, metricDefinition := range *metricDefinitions.Value {
			filteredMetricDefinitions = append(filteredMetricDefinitions, metricDefinition)
		}
		groupedMetrics := filterOnTimeGrain(filteredMetricDefinitions)
		for time, groupedMetricList := range groupedMetrics {
			// map azure metric definitions to client metrics
			dimMetrics := azure.GroupMetricsByAllDimensions(groupedMetricList)
			for dimension, mets := range dimMetrics {
				var dimensions []azure.Dimension
				if dimension != azure.NoDimension {
					dimensions = []azure.Dimension{{Name: dimension, Value: "*"}}
				}
				metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, mets, resource, resourceID, *namespace.Properties.MetricNamespaceName, dimensions, time)...)
			}
		}
	}
	return metrics, nil
}

// addMetricValues will map the metric values in a specific way for the storage metricset
func addMetricValues(event *mb.Event, metricValues common.MapStr) error {
	namespace, err := event.ModuleFields.GetValue("namespace")
	if err != nil {
		return errors.New("event namespace has not been set")
	}
	if i := retrieveServiceNamespace(namespace.(string)); i != "" {
		name := strings.TrimPrefix(i, "/")
		name = strings.TrimSuffix(name, "Services")
		for key, metric := range metricValues {
			event.MetricSetFields.Put(fmt.Sprintf("%s.%s", name, key), metric)
		}
	} else {
		for key, metric := range metricValues {
			event.MetricSetFields.Put(key, metric)
		}
	}
	return nil
}

// filterOnTimeGrain - some metrics do not support the default timegrain value so the closest supported timegrain will be selected
func filterOnTimeGrain(list []insights.MetricDefinition) map[string][]insights.MetricDefinition {
	var groupedList = make(map[string][]insights.MetricDefinition)
	for _, metric := range list {
		timegrain := retrieveSupportedMetricAvailability(*metric.MetricAvailabilities)
		if _, ok := groupedList[timegrain]; !ok {
			groupedList[timegrain] = make([]insights.MetricDefinition, 0)
		}
		groupedList[timegrain] = append(groupedList[timegrain], metric)
	}
	return groupedList
}

// retrieveSupportedMetricAvailability func will return the default timegrain if supported, else will return the next timegrain
func retrieveSupportedMetricAvailability(availabilities []insights.MetricAvailability) string {
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
