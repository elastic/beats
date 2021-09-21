// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
)

// Service interface for the azure monitor service and mock for testing
type Service interface {
	GetResourceDefinitionById(id string) (resources.GenericResource, error)
	GetResourceDefinitions(id []string, group []string, rType string, query string) ([]resources.GenericResourceExpanded, error)
	GetMetricDefinitions(resourceId string, namespace string) (insights.MetricDefinitionCollection, error)
	GetMetricNamespaces(resourceId string) (insights.MetricNamespaceCollection, error)
	GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, string, error)
}
