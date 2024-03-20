// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

const missingNamespace = "no metric definitions were found for resource %s and namespace %s. Verify if the namespace is spelled correctly or if it is supported by the resource in case."

// mapMetrics should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetrics(client *azure.Client, resources []*armresources.GenericResourceExpanded, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {
	var metrics []azure.Metric

	for _, resource := range resources {

		// We use this map to avoid calling the metrics definition function for the same namespace and same resource
		// multiple times.
		namespaceMetrics := make(map[string]armmonitor.MetricDefinitionCollection)

		for _, metric := range resourceConfig.Metrics {

			var err error

			metricDefinitions, exists := namespaceMetrics[metric.Namespace]
			if !exists {
				metricDefinitions, err = client.AzureMonitorService.GetMetricDefinitionsWithRetry(*resource.ID, metric.Namespace)
				if err != nil {
					return nil, err
				}
				namespaceMetrics[metric.Namespace] = metricDefinitions
			}

			if len(metricDefinitions.Value) == 0 {
				if metric.IgnoreUnsupported {
					client.Log.Infof(missingNamespace, *resource.ID, metric.Namespace)
					continue
				}
				return nil, fmt.Errorf("%s %s %s", missingNamespace, *resource.ID, metric.Namespace)
			}

			// validate metric names and filter on the supported metrics
			supportedMetricNames, err := filterMetricNames(*resource.ID, metric, metricDefinitions.Value)
			if err != nil {
				return nil, err
			}

			//validate aggregations and filter on supported aggregations
			metricGroups, err := filterOnSupportedAggregations(supportedMetricNames, metric, metricDefinitions.Value)
			if err != nil {
				return nil, err
			}

			// map dimensions
			var dim []azure.Dimension
			if len(metric.Dimensions) > 0 {
				for _, dimension := range metric.Dimensions {
					dim = append(dim, azure.Dimension(dimension))
				}
			}
			for key, metricGroup := range metricGroups {
				var metricNames []string
				for _, metricName := range metricGroup {
					metricNames = append(metricNames, *metricName.Name.Value)
				}
				metrics = append(metrics, client.CreateMetric(*resource.ID, "", metric.Namespace, metricNames, key, dim, metric.Timegrain))
			}
		}
	}

	return metrics, nil
}

// filterMetricNames func will verify if the metric names entered are valid and will also return the corresponding list of metrics
func filterMetricNames(resourceId string, metricConfig azure.MetricConfig, metricDefinitions []*armmonitor.MetricDefinition) ([]string, error) {
	var supportedMetricNames []string
	var unsupportedMetricNames []string
	// If users selected the wildcard option (*), we add
	// all the metric definitions to the supported metric.
	if strings.Contains(strings.Join(metricConfig.Name, " "), "*") {
		for _, definition := range metricDefinitions {
			supportedMetricNames = append(supportedMetricNames, *definition.Name.Value)
		}
	} else {
		// verify if configured metric names are valid, return log error event for the invalid ones, map only  the valid metric names
		supportedMetricNames, unsupportedMetricNames = filterConfiguredMetrics(metricConfig.Name, metricDefinitions)
		if len(unsupportedMetricNames) > 0 && !metricConfig.IgnoreUnsupported {
			return nil, fmt.Errorf("the metric names configured  %s are not supported for the resource %s and namespace %s",
				strings.Join(unsupportedMetricNames, ","), resourceId, metricConfig.Namespace)
		}
	}
	if len(supportedMetricNames) == 0 && !metricConfig.IgnoreUnsupported {
		return nil, fmt.Errorf("the metric names configured : %s are not supported for the resource %s and namespace %s ", strings.Join(metricConfig.Name, ","), resourceId, metricConfig.Namespace)
	}
	return supportedMetricNames, nil
}

// filterConfiguredMetrics will filter out any unsupported metrics based on the namespace selected
func filterConfiguredMetrics(selectedRange []string, allRange []*armmonitor.MetricDefinition) ([]string, []string) {
	var inRange []string
	var notInRange []string
	var allMetrics string
	for _, definition := range allRange {
		allMetrics += *definition.Name.Value + " "
	}
	for _, name := range selectedRange {
		if strings.Contains(allMetrics, name) {
			inRange = append(inRange, name)
		} else {
			notInRange = append(notInRange, name)
		}
	}
	return inRange, notInRange
}

// filterOnSupportedAggregations will verify if the aggregation values entered are supported and will also return the corresponding list of aggregations
func filterOnSupportedAggregations(
	metricNames []string,
	metricConfig azure.MetricConfig,
	metricDefinitions []*armmonitor.MetricDefinition,
) (map[string][]*armmonitor.MetricDefinition, error) {
	var supportedAggregations []string
	var unsupportedAggregations []string
	metricGroups := make(map[string][]*armmonitor.MetricDefinition)
	metricDefs := getMetricDefinitionsByNames(metricDefinitions, metricNames)

	if len(metricConfig.Aggregations) == 0 {
		for _, metricDef := range metricDefs {
			metricGroups[string(*metricDef.PrimaryAggregationType)] = append(metricGroups[string(*metricDef.PrimaryAggregationType)], metricDef)
		}
	} else {
		supportedAggregations, unsupportedAggregations = filterAggregations(metricConfig.Aggregations, metricDefs)
		if len(unsupportedAggregations) > 0 {
			return nil, fmt.Errorf("the aggregations configured : %s are not supported for some of the metrics selected %s ",
				strings.Join(unsupportedAggregations, ","), strings.Join(metricNames, ","))
		}
		if len(supportedAggregations) == 0 {
			return nil, fmt.Errorf("no aggregations were found based on the aggregation values configured or supported between the metrics : %s",
				strings.Join(metricNames, ","))
		}
		key := strings.Join(supportedAggregations, ",")
		metricGroups[key] = append(metricGroups[key], metricDefs...)
	}
	return metricGroups, nil
}

// filterAggregations will filter out any unsupported aggregations based on the metrics selected
func filterAggregations(selectedRange []string, metrics []*armmonitor.MetricDefinition) ([]string, []string) {
	var difference []string
	var supported = []string{"Average", "Maximum", "Minimum", "Count", "Total"}

	for _, metric := range metrics {
		var metricSupported []string
		for _, agg := range metric.SupportedAggregationTypes {
			metricSupported = append(metricSupported, string(*agg))
		}
		supported, _ = intersections(metricSupported, supported)
	}
	if len(selectedRange) != 0 {
		supported, difference = intersections(supported, selectedRange)
	}
	return supported, difference
}

// filter is a helper method, will filter out strings not part of a slice
func filter(src []string) []string {
	var res []string
	for _, s := range src {
		newStr := strings.Join(res, " ")
		if !strings.Contains(newStr, s) {
			res = append(res, s)
		}
	}
	return res
}

// intersections is a helper method, will compare 2 slices and return their intersection and difference records
func intersections(supported, selected []string) ([]string, []string) {
	var intersection []string
	var difference []string
	str1 := strings.Join(filter(supported), " ")
	for _, s := range filter(selected) {
		if strings.Contains(str1, s) {
			intersection = append(intersection, s)
		} else {
			difference = append(difference, s)
		}
	}
	return intersection, difference
}

// getMetricDefinitionsByNames is a helper method, will compare 2 slices and return their intersection
func getMetricDefinitionsByNames(metricDefs []*armmonitor.MetricDefinition, names []string) []*armmonitor.MetricDefinition {
	var metrics []*armmonitor.MetricDefinition
	for _, def := range metricDefs {
		for _, supportedName := range names {
			if *def.Name.Value == supportedName {
				metrics = append(metrics, def)
			}
		}
	}
	return metrics
}
