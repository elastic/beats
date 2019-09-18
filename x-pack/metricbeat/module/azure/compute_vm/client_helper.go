// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm

import (
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"

	"github.com/pkg/errors"
)

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetric(client *azure.Client, metric azure.MetricConfig, resource resources.GenericResource) ([]azure.Metric, error) {
	var metrics []azure.Metric
	//check if no metric names are configured
	if metric.Name == nil {
		return nil, nil
	}
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
		// map azure metric definitions to client metrics
		metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, *metricDefinitions.Value, resource, *namespace.Properties.MetricNamespaceName, nil, azure.DefaultTimeGrain)...)
	}
	return metrics, nil
}
