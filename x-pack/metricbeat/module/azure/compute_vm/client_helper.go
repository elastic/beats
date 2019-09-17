// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

const (
	noDimension      = "none"
	defaultTimeGrain = "PT5M"
)

// InitResources returns the list of resources and maps them.
func InitResources(client *azure.Client, report mb.ReporterV2) error {
	if len(client.Config.Resources) == 0 {
		return errors.New("no resource options defined")
	}
	// check if refresh interval has been set and if it has expired
	if !client.Resources.Expired() {
		return nil
	}
	var metrics []azure.Metric
	for _, resource := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.ID, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve resources")
			client.LogError(report, err)
			continue
		}
		if len(resourceList.Values()) == 0 {
			err = errors.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.ID, resource.Group, resource.Type, resource.Query)
			client.LogError(report, err)
			continue
		}
		for _, res := range resourceList.Values() {
			namespaces, err := client.AzureMonitorService.GetMetricNamespaces(*res.ID)
			if err != nil {
				return errors.Wrapf(err, "no metric namespaces were found for resource %s.", *res.ID)
			}
			for _, namespace := range *namespaces.Value {
				// get all metric definitions supported by the namespace provided
				metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(*res.ID, *namespace.Name)
				if err != nil {
					return errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s.", resource.ID, *namespace.Name)
				}
				if len(*metricDefinitions.Value) == 0 {
					return errors.Errorf("no metric definitions were found for resource %s and namespace %s.", resource.ID, *namespace.Name)
				}
				// map azure metric definitions to client metrics
				metrics = append(metrics, mapMetric(client, res, metricDefinitions, *namespace.Name)...)
			}
		}
	}

	// users could add or remove resources while metricbeat is running so we could encounter the situation where resources are unavailable, we log and create an event if this is the case (see above)
	// but we return an error when absolutely no resources are found
	if len(metrics) == 0 {
		return errors.New("no resources were found based on all the configurations options entered")
	}
	client.Resources.Metrics = metrics
	return nil
}

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetric(client *azure.Client, resource resources.GenericResource, metricDefinitions insights.MetricDefinitionCollection, namespace string) []azure.Metric {
	var metrics []azure.Metric
	groupByDimensions := make(map[string][]insights.MetricDefinition)
	// some of the metrics do not support the vmname dimension (or any), we will separate those in a different api call

	for _, metric := range *metricDefinitions.Value {
		if len(*metric.Dimensions) == 0 {
			groupByDimensions["none"] = append(groupByDimensions[noDimension], metric)
		} else {
			for _, dim := range *metric.Dimensions {
				groupByDimensions[*dim.Value] = append(groupByDimensions[*dim.Value], metric)
			}
		}
	}
	if len(groupByDimensions) > 0 {
		for key, groupedMetrics := range groupByDimensions {
			var dim []azure.Dimension
			if key != noDimension {
				dim = []azure.Dimension{{Name: key, Value: "*"}}
			}
			metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, groupedMetrics, resource, namespace, dim, defaultTimeGrain)...)

		}
	}
	return metrics
}
