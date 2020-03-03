// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

// mapMetrics should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetrics(client *azure.Client, resources []resources.GenericResource, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {
	var metrics []azure.Metric
	if len(resourceConfig.Metrics) == 0 {
		return nil, nil
	}
	for _, resource := range resources {
		// return resource size
		resourceSize := mapResourceSize(resource, client)
		// return all namespaces supported for this resource
		namespaces, err := client.AzureMonitorService.GetMetricNamespaces(*resource.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "no metric namespaces were found for resource %s", *resource.ID)
		}
		for _, namespace := range *namespaces.Value {
			// get all metric definitions supported by the namespace provided
			metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(*resource.ID, *namespace.Properties.MetricNamespaceName)
			if err != nil {
				return nil, errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s.", *resource.ID, *namespace.Properties.MetricNamespaceName)
			}
			if len(*metricDefinitions.Value) == 0 {
				return nil, errors.Errorf("no metric definitions were found for resource %s and namespace %s.", *resource.ID, *namespace.Properties.MetricNamespaceName)
			}
			var filteredMetricDefinitions []insights.MetricDefinition
			for _, metricDefinition := range *metricDefinitions.Value {
				filteredMetricDefinitions = append(filteredMetricDefinitions, metricDefinition)
			}
			// map azure metric definitions to client metrics
			metrics = append(metrics, client.MapMetricByPrimaryAggregation(filteredMetricDefinitions, resource, "", resourceSize, *namespace.Properties.MetricNamespaceName, nil, azure.DefaultTimeGrain)...)
		}
	}
	return metrics, nil
}

// mapResourceSize func will try to map if existing the resource size
func mapResourceSize(resource resources.GenericResource, client *azure.Client) string {
	if resource.Sku != nil && resource.Sku.Name != nil {
		return *resource.Sku.Name
	}
	if resource.Sku == nil && resource.Properties == nil {
		expandedResource, err := client.AzureMonitorService.GetResourceDefinitionById(*resource.ID)
		if err != nil {
			client.Log.Error(err, "could not retrieve the resource details by resource ID %s", *resource.ID)
			return ""
		}
		if expandedResource.Properties != nil {
			if properties, ok := expandedResource.Properties.(map[string]interface{}); ok {
				if hardware, ok := properties["hardwareProfile"]; ok {
					if vmSize, ok := hardware.(map[string]interface{})["vmSize"]; ok {
						return vmSize.(string)
					}
				}
			}
		}
	}
	return ""
}
