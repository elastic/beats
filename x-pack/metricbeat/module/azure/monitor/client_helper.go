// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/pkg/errors"
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
			for _, metric := range resource.Metrics {

				met, err := mapMetric(client, metric, res)
				if err != nil {
					client.LogError(report, err)
					continue
				}
				metrics = append(metrics, met...)
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
func mapMetric(client *azure.Client, metric azure.MetricConfig, resource resources.GenericResource) ([]azure.Metric, error) {
	var metrics []azure.Metric
	// get all metrics supported by the namespace provided
	metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(*resource.ID, metric.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s.", *resource.ID, metric.Namespace)
	}
	if len(*metricDefinitions.Value) == 0 {
		return nil, errors.Errorf("no metric definitions were found for resource %s and namespace %s.", *resource.ID, metric.Namespace)
	}

	// validate metric names
	// check if all metric names are selected (*)
	var supportedMetricNames []string
	var unsupportedMetricNames []string
	if azure.StringInSlice("*", metric.Name) {
		for _, definition := range *metricDefinitions.Value {
			supportedMetricNames = append(supportedMetricNames, *definition.Name.Value)
		}
	} else {
		// verify if configured metric names are valid, return log error event for the invalid ones, map only  the valid metric names
		supportedMetricNames, unsupportedMetricNames = filterMetrics(metric.Name, *metricDefinitions.Value)
		if len(unsupportedMetricNames) > 0 {
			return nil, errors.Errorf("none of metric names configured are supported by the resources found : %s are not supported for namespace %s ",
				strings.Join(unsupportedMetricNames, ","), metric.Namespace)
		}
	}
	if len(supportedMetricNames) == 0 {
		return nil, errors.Errorf("the metric names configured : %s are not supported for namespace %s ", strings.Join(metric.Name, ","), metric.Namespace)
	}
	//validate aggregations and filter on supported ones
	var supportedAggregations []string
	var unsupportedAggregations []string
	metricGroups := make(map[string][]insights.MetricDefinition)
	metricDefs := getMetricDefinitionsByNames(*metricDefinitions.Value, supportedMetricNames)

	if len(metric.Aggregations) == 0 {
		for _, metricDef := range metricDefs {
			metricGroups[string(metricDef.PrimaryAggregationType)] = append(metricGroups[string(metricDef.PrimaryAggregationType)], metricDef)
		}
	} else {
		supportedAggregations, unsupportedAggregations = filterAggregations(metric.Aggregations, metricDefs)
		if len(unsupportedAggregations) > 0 {
			return nil, errors.Errorf("the aggregations configured : %s are not supported for some of the metrics selected %s ",
				strings.Join(unsupportedAggregations, ","), strings.Join(supportedMetricNames, ","))
		}
		if len(supportedAggregations) == 0 {
			return nil, errors.Errorf("no aggregations were found based on the aggregation values configured or supported between the metrics : %s",
				strings.Join(supportedMetricNames, ","))
		}
		metricGroups[strings.Join(supportedAggregations, ",")] = append(metricGroups[strings.Join(supportedMetricNames, ",")], metricDefs...)
	}

	// map dimensions
	var dim []azure.Dimension
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			dim = append(dim, azure.Dimension{Name: dimension.Name, Value: dimension.Value})
		}
	}

	for key, metricGroup := range metricGroups {
		var metricNames []string
		for _, metricName := range metricGroup {
			metricNames = append(metricNames, *metricName.Name.Value)
		}
		metrics = append(metrics, client.CreateMetric(resource, metric.Namespace, metricNames, key, dim, metric.Timegrain))
	}
	return metrics, nil
}
