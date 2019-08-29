// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

// AzureMockService mock for the azure monitor services
type AzureMockService struct {
	mock.Mock
}

// GetResourceDefinitions is a mock function for the azure service
func (client *AzureMockService) GetResourceDefinitions(ID string, group string, rType string, query string) (resources.ListResultPage, error) {
	args := client.Called(ID, group, rType, query)
	return args.Get(0).(resources.ListResultPage), args.Error(1)
}

// GetMetricDefinitions is a mock function for the azure service
func (client *AzureMockService) GetMetricDefinitions(resourceID string, namespace string) (insights.MetricDefinitionCollection, error) {
	args := client.Called(resourceID, namespace)
	return args.Get(0).(insights.MetricDefinitionCollection), args.Error(1)
}

// GetMetricValues is a mock function for the azure service
func (client *AzureMockService) GetMetricValues(resourceID string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, error) {
	args := client.Called(resourceID, namespace)
	return args.Get(0).([]insights.Metric), args.Error(1)
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

func MockMetricDefinitions() *[]insights.MetricDefinition {
	metric1 := "TotalRequests"
	metric2 := "Capacity"
	metric3 := "BytesRead"
	defs := []insights.MetricDefinition{
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
	return &defs
}

func TestInitResources(t *testing.T) {
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := MockClient()
		mr := MockReporterV2{}
		err := client.InitResources(&mr)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client := MockClient()
		client.config = resourceQueryConfig
		m := &AzureMockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(resources.ListResultPage{}, errors.New("invalid resource query"))
		client.azureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.InitResources(&mr)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Equal(t, len(client.resources.metrics), 0)
		m.AssertExpectations(t)
	})
}
func TestMapMetric(t *testing.T) {
	resource := MockResource()
	metricDefinitions := insights.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	client := MockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(insights.MetricDefinitionCollection{}, errors.New("invalid resource ID"))
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
		assert.Equal(t, metric.names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, metric.aggregations, "Average,Count")
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
		assert.Equal(t, metric.names, []string{"TotalRequests"})
		assert.Equal(t, metric.aggregations, "Average,Total")
		assert.Equal(t, metric.dimensions, []Dimension{{name: "location", value: "West Europe"}})
		m.AssertExpectations(t)
	})
}

func TestGetMetricValues(t *testing.T) {
	client := MockClient()
	client.config = resourceIDConfig
	t.Run("return no error when no metric values are returned but log and send event", func(t *testing.T) {
		client.resources = ResourceConfiguration{
			metrics: []Metric{
				{
					namespace:    "namespace",
					names:        []string{"TotalRequests,Capacity"},
					aggregations: "Average,Total",
					dimensions:   []Dimension{{name: "location", value: "West Europe"}},
				},
			},
		}
		m := &AzureMockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]insights.Metric{}, errors.New("invalid parameters or no metrics found"))
		client.azureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.GetMetricValues(&mr)
		assert.Nil(t, err)
		assert.Equal(t, len(client.resources.metrics[0].values), 0)
		m.AssertExpectations(t)
	})
}
