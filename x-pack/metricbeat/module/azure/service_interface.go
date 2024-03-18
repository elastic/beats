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
	GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]armmonitor.Metric, string, error)
}
