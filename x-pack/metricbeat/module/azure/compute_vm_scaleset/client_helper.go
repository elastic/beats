// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compute_vm_scaleset

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

const (
	defaultVMScalesetNamespace = "Microsoft.Compute/virtualMachineScaleSets"
	customVMNamespace          = "Azure.VM.Windows.GuestMetrics"
	defaultVMDimension         = "VMName"
	customVMDimension          = "VirtualMachine"
	defaultSlotIDDimension     = "SlotId"
	defaultTimeGrain           = "PT5M"
)

var memoryMetrics = []string{"Memory\\Commit Limit", "Memory\\Committed Bytes", "Memory\\% Committed Bytes In Use", "Memory\\Available Bytes"}

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
		if len(resource.Group)>0 {
			resource.Type = defaultVMScalesetNamespace
		}
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
				return errors.Wrapf(err, "no metric namespaces were found for resource %s.", resource.ID)
			}
			for _, namespace := range *namespaces.Value {
				metricDefinitions, err := client.AzureMonitorService.GetMetricDefinitions(*res.ID, *namespace.Properties.MetricNamespaceName)
				if err != nil {
					return errors.Wrapf(err, "no metric definitions were found for resource %s and namespace %s.", *res.ID, *namespace.Properties.MetricNamespaceName)
				}
				if len(*metricDefinitions.Value) == 0 {
					return errors.Errorf("no metric definitions were found for resource %s and namespace %s.", *res.ID, *namespace.Properties.MetricNamespaceName)
				}
				// map azure metric definitions to client metrics
				metrics = append(metrics, mapMetric(client, res, metricDefinitions, *namespace.Properties.MetricNamespaceName)...)
			}
			// get all metric definitions supported by the namespace provided

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
	var vmNameMetricNames []insights.MetricDefinition
	var genericMetricNames []insights.MetricDefinition
	var slotMetricNames []insights.MetricDefinition
	var vmdim string
	if namespace == defaultVMScalesetNamespace {
		vmdim = defaultVMDimension
	} else if namespace == customVMNamespace {
		vmdim = customVMDimension
	}

	// some of the metrics do not support the vmname dimension (or any), we will separate those in a different api call
	for _, metric := range *metricDefinitions.Value {
		if (metric.Name.LocalizedValue!=nil && !strings.Contains(*metric.Name.LocalizedValue, "(Deprecated)")) || metric.Name.LocalizedValue== nil {
			if (namespace == customVMNamespace && azure.StringInSlice(*metric.Name.Value, memoryMetrics)) || namespace == defaultVMScalesetNamespace {
				if metric.Dimensions == nil || len(*metric.Dimensions) == 0 {
					genericMetricNames = append(genericMetricNames, metric)
				} else if containsDimension(vmdim, *metric.Dimensions) {
					vmNameMetricNames = append(vmNameMetricNames, metric)
				} else if containsDimension(defaultSlotIDDimension, *metric.Dimensions) {
					slotMetricNames = append(slotMetricNames, metric)
				}
			}
		}
	}
	if len(genericMetricNames) > 0 {

		metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, genericMetricNames, resource, namespace, nil, defaultTimeGrain)...)
	}
	if len(vmNameMetricNames) > 0 {
		metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, vmNameMetricNames, resource, namespace, []azure.Dimension{{Name: vmdim, Value: "*"}}, defaultTimeGrain)...)
	}
	if len(slotMetricNames) > 0 {
		metrics = append(metrics, azure.MapMetricByPrimaryAggregation(client, slotMetricNames, resource, namespace, []azure.Dimension{{Name: defaultSlotIDDimension, Value: "*"}}, defaultTimeGrain)...)
	}
	return metrics
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
