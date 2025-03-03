// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

// Service interface for the azure monitor service and mock for testing
type Service interface {
	GetResourceDefinitionById(id string) (armresources.GenericResource, error)
	GetResourceDefinitions(id []string, group []string, rType string, query string) ([]*armresources.GenericResourceExpanded, error)
	GetMetricDefinitionsWithRetry(resourceId string, namespace string) (armmonitor.MetricDefinitionCollection, error)
	GetMetricNamespaces(resourceId string) (armmonitor.MetricNamespaceCollection, error)
	// GetMetricValues returns the metric values for the given resource ID, namespace, timegrain, timespan, metricNames, aggregations and filter.
	//
	// If the timegrain is empty, the default timegrain for the metric is used and returned.
	GetMetricValues(
		resourceId string, // resourceId is the ID of the resource to query (e.g. "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}")
		namespace string, // namespace is the metric namespace to query (e.g. "Microsoft.Compute/virtualMachines")
		timegrain string, // timegrain is the timegrain to use for the metric query (e.g. "PT1M"); if empty, returns the default timegrain for the metric.
		timespan string, // timespan is the time interval to query (e.g. 2024-04-29T14:03:00Z/2024-04-29T14:04:00Z)
		metricNames []string, // metricNames is the list of metric names to query (e.g. ["ServiceApiLatency", "Availability"])
		aggregations string, // aggregations is the comma-separated list of aggregations to use for the metric query (e.g. "Average,Maximum,Minimum")
		filter string, // filter is the filter to query for dimensions (e.g. "ActivityType eq '*' AND ActivityName eq '*' AND StatusCode eq '*' AND StatusCodeClass eq '*'")
	) ([]armmonitor.Metric, string, error)
}
