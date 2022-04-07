// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/x-pack/metricbeat/module/azure"
)

func MockResourceExpanded() resources.GenericResourceExpanded {
	id := "123"
	name := "resourceName"
	location := "resourceLocation"
	rType := "resourceType"
	return resources.GenericResourceExpanded{
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
			PrimaryAggregationType:    insights.Average,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Maximum, insights.Count, insights.Total, insights.Average},
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric2},
			PrimaryAggregationType:    insights.Average,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric3},
			PrimaryAggregationType:    insights.Average,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
		},
	}
	return &defs
}

func TestMapMetric(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := insights.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(insights.MetricDefinitionCollection{}, errors.New("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []resources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []resources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "Capacity"}
		metricConfig.Aggregations = []string{"Average"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []resources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		assert.True(t, len(metrics) > 0)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		m.AssertExpectations(t)
	})
}

func TestFilterSConfiguredMetrics(t *testing.T) {
	selectedRange := []string{"TotalRequests", "Capacity", "CPUUsage"}
	intersection, difference := filterConfiguredMetrics(selectedRange, *MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"TotalRequests", "Capacity"})
	assert.Equal(t, difference, []string{"CPUUsage"})
}

func TestFilterAggregations(t *testing.T) {
	selectedRange := []string{"Average", "Minimum"}
	intersection, difference := filterAggregations(selectedRange, *MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"Average"})
	assert.Equal(t, difference, []string{"Minimum"})
}

func TestFilter(t *testing.T) {
	str := []string{"hello", "test", "goodbye", "test"}
	filtered := filter(str)
	assert.Equal(t, len(filtered), 3)
}

func TestIntersections(t *testing.T) {
	firstStr := []string{"test1", "test2", "test2", "test3"}
	sercondStr := []string{"test4", "test5", "test2", "test5", "test3"}
	intersection, difference := intersections(firstStr, sercondStr)
	assert.Equal(t, intersection, []string{"test2", "test3"})
	assert.Equal(t, difference, []string{"test4", "test5"})

	firstStr = []string{"test1", "test2", "test2", "test3"}
	sercondStr = []string{"test4", "test5", "test5"}
	intersection, difference = intersections(firstStr, sercondStr)
	assert.Equal(t, len(intersection), 0)
	assert.Equal(t, difference, []string{"test4", "test5"})
}

func TestGetMetricDefinitionsByNames(t *testing.T) {
	metrics := []string{"TotalRequests", "CPUUsage"}
	result := getMetricDefinitionsByNames(*MockMetricDefinitions(), metrics)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, *result[0].Name.Value, "TotalRequests")
}
