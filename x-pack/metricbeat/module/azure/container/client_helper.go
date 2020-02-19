// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package container

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

var retquiredDimension = "status"

// mapMetrics should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetrics(client *azure.Client, resources []resources.GenericResource, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {
	var metrics []azure.Metric
	if len(resourceConfig.Metrics) == 0 {
		return nil, nil
	}
	for _, resource := range resources {
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

			groupedMetrics := make(map[string][]insights.MetricDefinition)
			for _, metricName := range filteredMetricDefinitions {
				if *metricName.IsDimensionRequired == false {
					groupedMetrics[azure.NoDimension] = append(groupedMetrics[azure.NoDimension], metricName)
				} else if azure.ContainsDimension(retquiredDimension, *metricName.Dimensions) {
					groupedMetrics[retquiredDimension] = append(groupedMetrics[retquiredDimension], metricName)
				}
			}
			for key, metricGroup := range groupedMetrics {
				var metricNameList []string
				for _, metricName := range metricGroup {
					metricNameList = append(metricNameList, *metricName.Name.Value)
				}
				var dimensions []azure.Dimension
				if key != azure.NoDimension {
					dimensions = []azure.Dimension{{Name: key, Value: "*"}}
				}
				// map azure metric definitions to client metrics
				metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, metricGroup, resource, "", *namespace.Properties.MetricNamespaceName, dimensions, azure.DefaultTimeGrain)...)
			}
		}
	}
	return metrics, nil
}
