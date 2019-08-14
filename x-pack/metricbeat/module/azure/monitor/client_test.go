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

// GetResourceById is a mock function for the azure service
func (client *AzureMockService) GetResourceById(resourceID string) (resources.GenericResource, error) {
	args := client.Called(resourceID)
	return args.Get(0).(resources.GenericResource), args.Error(1)
}

// GetResourcesByResourceGroup is a mock function for the azure service
func (client AzureMockService) GetResourcesByResourceGroup(resourceGroup string, resourceType string) ([]resources.GenericResource, error) {
	args := client.Called(resourceGroup, resourceType)
	return args.Get(0).([]resources.GenericResource), args.Error(1)
}

// GetResourcesByResourceQuery is a mock function for the azure service
func (client AzureMockService) GetResourcesByResourceQuery(resourceQuery string) ([]resources.GenericResource, error) {
	args := client.Called(resourceQuery)
	return args.Get(0).([]resources.GenericResource), args.Error(1)
}

// GetMetricDefinitions is a mock function for the azure service
func (client AzureMockService) GetMetricDefinitions(resourceID string, namespace string) ([]insights.MetricDefinition, error) {
	args := client.Called(resourceID, namespace)
	return args.Get(0).([]insights.MetricDefinition), args.Error(1)
}

// GetMetricValues is a mock function for the azure service
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
				Type:  "typeName",
				Metrics: []azure.MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
	resourceQueryConfig = azure.Config{
		Resources: []azure.ResourceConfig{
			{
				Query: "query",
				Metrics: []azure.MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
)

func MockClient() *Client {
	azureMockService := new(AzureMockService)
	client := &Client{
		azureMonitorService: azureMockService,
		config:              invalidConfig,
		log:                 logp.NewLogger("test azure monitor"),
	}
	return client
}

func MockResource() resources.GenericResource {
	id := "123"
	name := "resourceName"
	location := "resourceLocation"
	rType := "resourceType"
	return resources.GenericResource{
		ID:       &id,
		Name:     &name,
		Location: &location,
		Type:     &rType,
	}
}

func MockMetricDefinitions() []insights.MetricDefinition {
	metric1 := "TotalRequests"
	metric2 := "Capacity"
	metric3 := "BytesRead"
	return []insights.MetricDefinition{
		{
			Name:                      &insights.LocalizableString{Value: &metric1},
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Maximum, insights.Count, insights.Total, insights.Average},
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric2},
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric3},
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
		},
	}
}

func TestInitResources(t *testing.T) {
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := MockClient()
		err := client.InitResources()
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return no error but log message when the resource id is invalid", func(t *testing.T) {
		client := MockClient()
		client.config = resourceIDConfig
		m := &AzureMockService{}
		m.On("GetResourceById", "123").Return(resources.GenericResource{}, errors.New("invalid resource ID"))
		client.azureMonitorService = m
		err := client.InitResources()
		assert.NoError(t, err)
		assert.Equal(t, len(client.resources.metrics), 0)
		m.AssertExpectations(t)
	})
	t.Run("return no error but log message when the resource group or type is invalid or no resources were found", func(t *testing.T) {
		client := MockClient()
		client.config = resourceGroupConfig
		m := &AzureMockService{}
		m.On("GetResourcesByResourceGroup", "groupName", "resourceType eq 'typeName'").Return([]resources.GenericResource{}, errors.New("invalid resource group"))
		client.azureMonitorService = m
		err := client.InitResources()
		assert.NoError(t, err)
		assert.Equal(t, len(client.resources.metrics), 0)
		m.AssertExpectations(t)
	})
	t.Run("return no error but log message when the resource query is invalid or no resources were found", func(t *testing.T) {
		client := MockClient()
		client.config = resourceQueryConfig
		m := &AzureMockService{}
		m.On("GetResourcesByResourceQuery", "query").Return([]resources.GenericResource{}, errors.New("invalid resource query"))
		client.azureMonitorService = m
		err := client.InitResources()
		assert.NoError(t, err)
		assert.Equal(t, len(client.resources.metrics), 0)
		m.AssertExpectations(t)
	})
}
func TestMapMetric(t *testing.T) {
	resource := MockResource()
	metricDefinitions := MockMetricDefinitions()
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	client := MockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return([]insights.MetricDefinition{}, errors.New("invalid resource ID"))
		client.azureMonitorService = m
		metric, err := client.mapMetric(metricConfig, resource)
		assert.NotNil(t, err)
		assert.Equal(t, metric, Metric{})
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(metricDefinitions, nil)
		client.azureMonitorService = m
		metricConfig.Name = []string{"*"}
		metric, err := client.mapMetric(metricConfig, resource)
		assert.Nil(t, err)
		assert.Equal(t, metric.resource.ID, "123")
		assert.Equal(t, metric.resource.Name, "resourceName")
		assert.Equal(t, metric.resource.Type, "resourceType")
		assert.Equal(t, metric.resource.Location, "resourceLocation")
		assert.Equal(t, metric.namespace, "namespace")
		assert.Equal(t, metric.names, "TotalRequests,Capacity,BytesRead")
		assert.Equal(t, metric.aggregations, "Count,Average")
		assert.Equal(t, metric.dimensions, []Dimension{{name: "location", value: "West Europe"}})
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(metricDefinitions, nil)
		client.azureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "CPU"}
		metricConfig.Aggregations = []string{"Average", "Total", "Minimum"}
		metric, err := client.mapMetric(metricConfig, resource)
		assert.Nil(t, err)
		assert.Equal(t, metric.resource.ID, "123")
		assert.Equal(t, metric.resource.Name, "resourceName")
		assert.Equal(t, metric.resource.Type, "resourceType")
		assert.Equal(t, metric.resource.Location, "resourceLocation")
		assert.Equal(t, metric.namespace, "namespace")
		assert.Equal(t, metric.names, "TotalRequests")
		assert.Equal(t, metric.aggregations, "Average,Total")
		assert.Equal(t, metric.dimensions, []Dimension{{name: "location", value: "West Europe"}})
		m.AssertExpectations(t)
	})
}

func TestGetMetricValues(t *testing.T) {
	client := MockClient()
	client.config = resourceIDConfig
	t.Run("return error when no metric values are returned", func(t *testing.T) {
		client.resources = ResourceConfiguration{
			metrics: []Metric{
				{
					namespace:    "namespace",
					names:        "TotalRequests,Capacity",
					aggregations: "Average,Total",
					dimensions:   []Dimension{{name: "location", value: "West Europe"}},
				},
			},
		}
		m := &AzureMockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]insights.Metric{}, errors.New("invalid parameters or no metrics found"))
		client.azureMonitorService = m
		err := client.GetMetricValues()
		assert.NotNil(t, err)
		assert.Equal(t, len(client.resources.metrics[0].values), 0)
		m.AssertExpectations(t)
	})
}
