// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

// AzureMockService mock for the azure monitor services
type AzureMockService struct {
	mock.Mock
}

func (client *AzureMockService) GetResourceById(resourceID string) (resources.GenericResource, error) {
	args := client.Called(resourceID)
	return args.Get(0).(resources.GenericResource), args.Error(1)
}
func (client AzureMockService) GetResourcesByResourceGroup(resourceGroup string, resourceType string) ([]resources.GenericResource, error) {
	args := client.Called(resourceGroup, resourceType)
	return args.Get(0).([]resources.GenericResource), args.Error(1)
}
func (client AzureMockService) GetResourcesByResourceQuery(resourceQuery string) ([]resources.GenericResource, error) {
	args := client.Called(resourceQuery)
	return args.Get(0).([]resources.GenericResource), args.Error(1)
}
func (client AzureMockService) GetMetricDefinitions(resourceID string, namespace string) ([]insights.MetricDefinition, error) {
	args := client.Called(resourceID, namespace)
	return args.Get(0).([]insights.MetricDefinition), args.Error(1)
}
func (client AzureMockService) GetMetricValues(resourceID string, namespace string, timespan string, metricNames string, aggregations string, filter string) ([]insights.Metric, error) {
	args := client.Called(resourceID, namespace)
	return args.Get(0).([]insights.Metric), args.Error(1)
}

var (
	invalidConfig    = azure.Config{}
	resourceIDConfig = azure.Config{
		Resources: []azure.ResourceConfig{
			{ID: "123",
				Metrics: []azure.MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
	resourceGroupConfig = azure.Config{
		Resources: []azure.ResourceConfig{
			{
				Group: "groupName",
				Type: "typeName",
				Metrics: []azure.MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
)

func NewMockClient() *Client {
	azureMockService := new(AzureMockService)
	client := &Client{
		azureMonitorService: azureMockService,
		config:              invalidConfig,
		log:                 logp.NewLogger("test azure monitor"),
	}
	return client
}

func TestInitResources(t *testing.T) {
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := NewMockClient()
		err := client.InitResources()
		assert.Error(t, err, "no resource options were configured")
	})

	t.Run("return no error but log message when the resource id is invalid", func(t *testing.T) {
		client := NewMockClient()
		client.config = resourceIDConfig
		m := &AzureMockService{}
		m.On("GetResourceById", "123").Return(resources.GenericResource{}, errors.New("invalid resource ID"))
		client.azureMonitorService = m
		err := client.InitResources()
		assert.NoError(t, err)
		assert.Equal(t, len(client.resourceConfig.metrics), 0)
		m.AssertExpectations(t)
	})

	t.Run("return no error but log message when the resource group or type is invalid or no resources were found", func(t *testing.T) {
		client := NewMockClient()
		client.config = resourceGroupConfig
		m := &AzureMockService{}
		m.On("GetResourcesByResourceGroup", "groupName", "resourceType eq 'typeName'").Return([]resources.GenericResource{}, errors.New("invalid resource group"))
		client.azureMonitorService = m
		err := client.InitResources()
		assert.NoError(t, err)
		assert.Equal(t, len(client.resourceConfig.metrics), 0)
		m.AssertExpectations(t)
	})
}
