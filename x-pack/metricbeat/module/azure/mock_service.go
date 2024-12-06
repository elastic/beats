// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// MockService mock for the azure monitor services
type MockService struct {
	mock.Mock
}

// GetResourceDefinitionById is a mock function for the azure service
func (client *MockService) GetResourceDefinitionById(id string) (armresources.GenericResource, error) {
	args := client.Called(id)
	return args.Get(0).(armresources.GenericResource), args.Error(1)
}

// GetResourceDefinitions is a mock function for the azure service
func (client *MockService) GetResourceDefinitions(id []string, group []string, rType string, query string) ([]*armresources.GenericResourceExpanded, error) {
	args := client.Called(id, group, rType, query)
	return args.Get(0).([]*armresources.GenericResourceExpanded), args.Error(1)
}

// GetMetricDefinitionsWithRetry is a mock function for the azure service
func (client *MockService) GetMetricDefinitionsWithRetry(resourceId string, namespace string) (armmonitor.MetricDefinitionCollection, error) {
	args := client.Called(resourceId, namespace)
	return args.Get(0).(armmonitor.MetricDefinitionCollection), args.Error(1)
}

// GetMetricNamespaces is a mock function for the azure service
func (client *MockService) GetMetricNamespaces(resourceId string) (armmonitor.MetricNamespaceCollection, error) {
	args := client.Called(resourceId)
	return args.Get(0).(armmonitor.MetricNamespaceCollection), args.Error(1)
}

// GetMetricValues is a mock function for the azure service
func (client *MockService) GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]armmonitor.Metric, string, error) {
	args := client.Called(resourceId, namespace, timegrain, timespan, metricNames, aggregations, filter)
	return args.Get(0).([]armmonitor.Metric), args.String(1), args.Error(2)
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
