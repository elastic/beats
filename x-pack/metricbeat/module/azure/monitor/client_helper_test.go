// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package monitor

import (
	"testing"

	"github.com/stretchr/testify/mock"

	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var (
	oneMinuteDuration    = `PT1M`
	thirtyMinuteDuration = `PT30M`
	oneHrDuration        = `PT1H`
	sixHrDuration        = `PT6H`
)

func MockResourceExpanded() *armresources.GenericResourceExpanded {
	id := "123"
	name := "resourceName"
	location := "resourceLocation"
	rType := "resourceType"

	return &armresources.GenericResourceExpanded{
		ID:       &id,
		Name:     &name,
		Location: &location,
		Type:     &rType,
	}
}

func MockMetricDefinitions() []*armmonitor.MetricDefinition {
	var (
		metric1 = "TotalRequests"
		metric2 = "Capacity"
		metric3 = "BytesRead"

		aggregationTypeAverage = armmonitor.AggregationTypeAverage
		aggregationTypeCount   = armmonitor.AggregationTypeCount
		aggregationTypeMinimum = armmonitor.AggregationTypeMinimum
		aggregationTypeMaximum = armmonitor.AggregationTypeMaximum
		aggregationTypeTotal   = armmonitor.AggregationTypeTotal
	)

	defs := []*armmonitor.MetricDefinition{
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric1},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeMaximum,
				&aggregationTypeCount,
				&aggregationTypeTotal,
				&aggregationTypeAverage,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &thirtyMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			}, // TODO: pick up here and add to other defs as well
		},
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric2},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeAverage,
				&aggregationTypeCount,
				&aggregationTypeMinimum,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &oneMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			},
		},
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric3},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeAverage,
				&aggregationTypeCount,
				&aggregationTypeMinimum,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &thirtyMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			},
		},
	}
	return defs
}

func TestMapMetricWithConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace",
		Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}},
		Timegrain:  oneHrDuration}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		assert.Equal(t, metrics[0].TimeGrain, oneHrDuration)
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "Capacity"}
		metricConfig.Aggregations = []string{"Average"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		assert.True(t, len(metrics) > 0)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		assert.Equal(t, metrics[0].TimeGrain, oneHrDuration)
		m.AssertExpectations(t)
	})
}

func TestMapMetricNoConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		// we should have two groups, one per first timegrain value
		assert.Len(t, metrics, 2)
		// this for loop with checks is necessary because the ordering of timegrains is non-deterministic
		// this is due to the fact that during mapMetrics, we are iterating over a map
		for _, metric := range metrics {
			switch metric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"Capacity"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			case thirtyMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"TotalRequests", "BytesRead"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			default:
				// cannot have any other cases
				t.FailNow()
			}
		}

		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "Capacity"}
		metricConfig.Aggregations = []string{"Average"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		assert.True(t, len(metrics) > 0)

		// we should have two groups, one per first timegrain value
		assert.Len(t, metrics, 2)
		// this for loop with checks is necessary because the ordering of timegrains is non-deterministic
		// this is due to the fact that during mapMetrics, we are iterating over a map
		for _, metric := range metrics {
			switch metric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"Capacity"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			case thirtyMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"TotalRequests"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			default:
				// cannot have any other cases
				t.FailNow()
			}
		}

		m.AssertExpectations(t)
	})
}

func TestFilterSConfiguredMetrics(t *testing.T) {
	selectedRange := []string{"TotalRequests", "Capacity", "CPUUsage"}
	intersection, difference := filterConfiguredMetrics(selectedRange, MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"TotalRequests", "Capacity"})
	assert.Equal(t, difference, []string{"CPUUsage"})
}

func TestFilterAggregations(t *testing.T) {
	selectedRange := []string{"Average", "Minimum"}
	intersection, difference := filterAggregations(selectedRange, MockMetricDefinitions())
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
	result := getMetricDefinitionsByNames(MockMetricDefinitions(), metrics)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, *result[0].Name.Value, "TotalRequests")
}
