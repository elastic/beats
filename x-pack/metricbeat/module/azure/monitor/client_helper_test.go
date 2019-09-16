// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
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
		client := azure.NewMockClient()
		mr := azure.MockReporterV2{}
		err := InitResources(client, &mr)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client := azure.NewMockClient()
		client.Config = resourceQueryConfig
		m := &azure.AzureMockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(resources.ListResultPage{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := azure.MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := InitResources(client, &mr)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Equal(t, len(client.Resources.Metrics), 0)
		m.AssertExpectations(t)
	})
}

func TestMapMetric(t *testing.T) {
	resource := MockResource()
	metricDefinitions := insights.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	client := azure.NewMockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(insights.MetricDefinitionCollection{}, errors.New("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetric(client, metricConfig, resource)
		assert.NotNil(t, err)
		assert.Equal(t, metric, azure.Metric{})
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		metrics, err := mapMetric(client, metricConfig, resource)
		assert.Nil(t, err)
		assert.Equal(t, metrics[0].Resource.ID, "123")
		assert.Equal(t, metrics[0].Resource.Name, "resourceName")
		assert.Equal(t, metrics[0].Resource.Type, "resourceType")
		assert.Equal(t, metrics[0].Resource.Location, "resourceLocation")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, metrics[0].Aggregations, "Average,Count")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.AzureMockService{}
		m.On("GetMetricDefinitions", "123", metricConfig.Namespace).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "CPU"}
		metricConfig.Aggregations = []string{"Average", "Total", "Minimum"}
		metrics, err := mapMetric(client, metricConfig, resource)
		assert.Nil(t, err)

		assert.True(t, len(metrics) > 0)
		assert.Equal(t, metrics[0].Resource.ID, "123")
		assert.Equal(t, metrics[0].Resource.Name, "resourceName")
		assert.Equal(t, metrics[0].Resource.Type, "resourceType")
		assert.Equal(t, metrics[0].Resource.Location, "resourceLocation")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests"})
		assert.Equal(t, metrics[0].Aggregations, "Average,Total")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		m.AssertExpectations(t)
	})
}
