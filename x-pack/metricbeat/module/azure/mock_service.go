// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
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
	if res, ok := args.Get(0).(armresources.GenericResource); ok {
		return res, args.Error(1)
	}
	return armresources.GenericResource{}, fmt.Errorf("error casting to armresources.GenericResource")
}

// GetResourceDefinitions is a mock function for the azure service
func (client *MockService) GetResourceDefinitions(id []string, group []string, rType string, query string) ([]*armresources.GenericResourceExpanded, error) {
	args := client.Called(id, group, rType, query)
	if res, ok := args.Get(0).([]*armresources.GenericResourceExpanded); ok {
		return res, args.Error(1)
	}
	return nil, fmt.Errorf("error casting to []*armresources.GenericResourceExpanded")
}

// GetMetricDefinitionsWithRetry is a mock function for the azure service
func (client *MockService) GetMetricDefinitionsWithRetry(resourceId string, namespace string) (armmonitor.MetricDefinitionCollection, error) {
	args := client.Called(resourceId, namespace)
	if res, ok := args.Get(0).(armmonitor.MetricDefinitionCollection); ok {
		return res, args.Error(1)
	}
	return armmonitor.MetricDefinitionCollection{}, fmt.Errorf("error casting to armmonitor.MetricDefinitionCollection")
}

// GetMetricNamespaces is a mock function for the azure service
func (client *MockService) GetMetricNamespaces(resourceId string) (armmonitor.MetricNamespaceCollection, error) {
	args := client.Called(resourceId)
	if res, ok := args.Get(0).(armmonitor.MetricNamespaceCollection); ok {
		return res, args.Error(1)
	}
	return armmonitor.MetricNamespaceCollection{}, fmt.Errorf("error casting to armmonitor.MetricNamespaceCollection")
}

// GetMetricValues is a mock function for the azure service
func (client *MockService) GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]armmonitor.Metric, string, error) {
	args := client.Called(resourceId, namespace, timegrain, timespan, metricNames, aggregations, filter)
	if res, ok := args.Get(0).([]armmonitor.Metric); ok {
		return res, args.String(1), args.Error(2)
	}
	return nil, "", fmt.Errorf("error casting to []armmonitor.Metric")
}

// QueryResources is a mock function for the azure service
func (client *MockService) QueryResources(
	resourceIDs []string,
	subscriptionID string,
	namespace string,
	timegrain string,
	startTime string,
	endTime string,
	metricNames []string,
	aggregations string,
	filter string,
	location string) ([]azmetrics.MetricData, error) {

	args := client.Called(resourceIDs, subscriptionID, namespace, timegrain, startTime, endTime, metricNames, aggregations, filter, location)
	if res, ok := args.Get(0).([]azmetrics.MetricData); ok {
		return res, args.Error(1)
	}
	return nil, fmt.Errorf("error casting to []azmetrics.MetricData")
}

// MockReporterV2 mock implementation for testing purposes
type MockReporterV2 struct {
	mock.Mock
}

// Event function is mock implementation for testing purposes
func (reporter *MockReporterV2) Event(event mb.Event) bool {
	args := reporter.Called(event)
	if res, ok := args.Get(0).(bool); ok {
		return res
	}
	return false
}

// Error is mock implementation for testing purposes
func (reporter *MockReporterV2) Error(err error) bool {
	args := reporter.Called(err)
	if res, ok := args.Get(0).(bool); ok {
		return res
	}
	return false
}
