// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v8/metricbeat/mb"
)

// MockService mock for the azure monitor services
type MockService struct {
	mock.Mock
}

// GetResourceDefinitionById is a mock function for the azure service
func (client *MockService) GetResourceDefinitionById(id string) (resources.GenericResource, error) {
	args := client.Called(id)
	return args.Get(0).(resources.GenericResource), args.Error(1)
}

// GetResourceDefinitions is a mock function for the azure service
func (client *MockService) GetResourceDefinitions(id []string, group []string, rType string, query string) ([]resources.GenericResourceExpanded, error) {
	args := client.Called(id, group, rType, query)
	return args.Get(0).([]resources.GenericResourceExpanded), args.Error(1)
}

// GetMetricDefinitions is a mock function for the azure service
func (client *MockService) GetMetricDefinitions(resourceId string, namespace string) (insights.MetricDefinitionCollection, error) {
	args := client.Called(resourceId, namespace)
	return args.Get(0).(insights.MetricDefinitionCollection), args.Error(1)
}

// GetMetricNamespaces is a mock function for the azure service
func (client *MockService) GetMetricNamespaces(resourceId string) (insights.MetricNamespaceCollection, error) {
	args := client.Called(resourceId)
	return args.Get(0).(insights.MetricNamespaceCollection), args.Error(1)
}

// GetMetricValues is a mock function for the azure service
func (client *MockService) GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, string, error) {
	args := client.Called(resourceId, namespace)
	return args.Get(0).([]insights.Metric), args.String(1), args.Error(2)
}

// MockReporterV2 mock implementation for testing purposes
type MockReporterV2 struct {
	mock.Mock
}

// Event function is mock implementation for testing purposes
func (reporter *MockReporterV2) Event(event mb.Event) bool {
	args := reporter.Called(event)
	return args.Get(0).(bool)
}

// Error is mock implementation for testing purposes
func (reporter *MockReporterV2) Error(err error) bool {
	args := reporter.Called(err)
	return args.Get(0).(bool)
}
