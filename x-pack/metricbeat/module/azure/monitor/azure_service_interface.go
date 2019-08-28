// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
)

// AzureService interface for the azure monitor service and mock for testing
type AzureService interface {
	GetResourceDefinitions(ID string, group string, rType string, query string) (resources.ListResultPage, error)
	GetMetricDefinitions(resourceID string, namespace string) (insights.MetricDefinitionCollection, error)
	GetMetricValues(resourceID string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, error)
}
