// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm_scaleset

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

const (
	defaultVMDimension     = "VMName"
	customVMDimension      = "VirtualMachine"
	defaultSlotIDDimension = "SlotId"
)

// mapMetrics should validate and map the metric related configuration to relevant azure monitor api parameters
func mapMetrics(client *azure.Client, resources []resources.GenericResource, resourceConfig azure.ResourceConfig) ([]azure.Metric, error) {
	var metrics []azure.Metric
	for _, resource := range resources {
		// return resource size
		resourceSize := mapResourceSize(resource)
		for _, metric := range resourceConfig.Metrics {
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
				continue
			}
			groupedMetrics := make(map[string][]insights.MetricDefinition)
			var vmdim string
			if metric.Namespace == defaultVMScalesetNamespace {
				vmdim = defaultVMDimension
			} else if metric.Namespace == customVMNamespace {
				vmdim = customVMDimension
			}
			for _, metricName := range supportedMetricNames {
				if metricName.Dimensions == nil || len(*metricName.Dimensions) == 0 {
					groupedMetrics[azure.NoDimension] = append(groupedMetrics[azure.NoDimension], metricName)
				} else if azure.ContainsDimension(vmdim, *metricName.Dimensions) {
					groupedMetrics[vmdim] = append(groupedMetrics[vmdim], metricName)
				} else if azure.ContainsDimension(defaultSlotIDDimension, *metricName.Dimensions) {
					groupedMetrics[defaultSlotIDDimension] = append(groupedMetrics[defaultSlotIDDimension], metricName)
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
				metrics = append(metrics, client.MapMetricByPrimaryAggregation(metricGroup, resource, "", resourceSize, metric.Namespace, dimensions, azure.DefaultTimeGrain)...)
			}
		}
	}
	return metrics, nil
}

// mapResourceSize func will try to map if existing the resource size, for the vmss it seems that SKU is populated and resource size is mapped in the name
func mapResourceSize(resource resources.GenericResource) string {
	if resource.Sku != nil && resource.Sku.Name != nil {
		return *resource.Sku.Name
	}
	return ""
}
