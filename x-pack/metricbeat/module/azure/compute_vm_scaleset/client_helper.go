// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm_scaleset

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"

	"github.com/pkg/errors"
)

const (
	defaultVMDimension     = "VMName"
	customVMDimension      = "VirtualMachine"
	defaultSlotIDDimension = "SlotId"
	defaultTimeGrain       = "PT5M"
	noDimension            = "none"
)

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetric(client *azure.Client, metric azure.MetricConfig, resource resources.GenericResource) ([]azure.Metric, error) {
	var metrics []azure.Metric
	metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(*resource.ID, metric.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s", *resource.ID, metric.Namespace)
	}
	if len(*metricDefinitions.Value) == 0 && metric.Namespace != customVMNamespace {
		return nil, errors.Errorf("no metric definitions were found for resource %s and namespace %s.", *resource.ID, metric.Namespace)
	}
	var supportedMetricNames []insights.MetricDefinition
	if strings.Contains(strings.Join(metric.Name, " "), "*") {
		for _, definition := range *metricDefinitions.Value {
			supportedMetricNames = append(supportedMetricNames, definition)
		}
	} else {
		// verify if configured metric names are valid, return log error event for the invalid ones, map only  the valid metric names
		for _, name := range metric.Name {
			for _, metricDefinition := range *metricDefinitions.Value {
				if name == *metricDefinition.Name.Value {
					supportedMetricNames = append(supportedMetricNames, metricDefinition)
				}
			}
		}
	}
	if len(supportedMetricNames) == 0 {
		return nil, nil
	}
	groupedMetrics := make(map[string][]insights.MetricDefinition)
	var vmdim string
	if metric.Namespace == defaultVMScalesetNamespace {
		vmdim = defaultVMDimension
	} else if metric.Namespace == customVMNamespace {
		vmdim = customVMDimension
	}
	for _, metricName := range supportedMetricNames {
		if (metricName.Name.LocalizedValue != nil && !strings.Contains(*metricName.Name.LocalizedValue, "(Deprecated)")) || metricName.Name.LocalizedValue == nil {
			if metricName.Dimensions == nil || len(*metricName.Dimensions) == 0 {
				groupedMetrics[noDimension] = append(groupedMetrics[noDimension], metricName)
			} else if containsDimension(vmdim, *metricName.Dimensions) {
				groupedMetrics[vmdim] = append(groupedMetrics[vmdim], metricName)
			} else if containsDimension(defaultSlotIDDimension, *metricName.Dimensions) {
				groupedMetrics[defaultSlotIDDimension] = append(groupedMetrics[defaultSlotIDDimension], metricName)
			}
		}
	}
	for key, metricGroup := range groupedMetrics {
		var metricNameList []string
		for _, metricName := range metricGroup {
			metricNameList = append(metricNameList, *metricName.Name.Value)
		}
		var dimensions []azure.Dimension
		if key != noDimension {
			dimensions = []azure.Dimension{{Name: vmdim, Value: "*"}}
		}
		metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, metricGroup, resource, metric.Namespace, dimensions, defaultTimeGrain)...)
	}

	return metrics, nil
}

// containsDimension will check if the dimension value is found in the list
func containsDimension(dimension string, dimensions []insights.LocalizableString) bool {
	for _, dim := range dimensions {
		if *dim.Value == dimension {
			return true
		}
	}
	return false
}
