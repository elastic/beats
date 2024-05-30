// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

var (
	time1         = "PT1M"
	time2         = "PT5M"
	time3         = "PT1H"
	availability1 = []*armmonitor.MetricAvailability{
		{TimeGrain: &time1},
		{TimeGrain: &time2},
	}
	availability2 = []*armmonitor.MetricAvailability{
		{TimeGrain: &time3},
	}
	availability3 = []*armmonitor.MetricAvailability{
		{TimeGrain: &time1},
		{TimeGrain: &time3},
	}
)

func MockResource() *armresources.GenericResourceExpanded {
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

func MockNamespace() armmonitor.MetricNamespaceCollection {
	name := "namespace"
	property := armmonitor.MetricNamespaceName{
		MetricNamespaceName: &name,
	}
	namespace := &armmonitor.MetricNamespace{
		Name:       &name,
		Properties: &property,
	}

	list := []*armmonitor.MetricNamespace{namespace}

	return armmonitor.MetricNamespaceCollection{
		Value: list,
	}
}

func MockMetricDefinitions() []*armmonitor.MetricDefinition {
	var (
		metric1 = "TotalRequests"
		metric2 = "Capacity"

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
			MetricAvailabilities:   availability1,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeMaximum,
				&aggregationTypeCount,
				&aggregationTypeTotal,
				&aggregationTypeAverage,
			},
		},
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric2},
			PrimaryAggregationType: &aggregationTypeAverage,
			MetricAvailabilities:   availability2,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeAverage,
				&aggregationTypeCount,
				&aggregationTypeMinimum,
			},
		},
	}

	return defs
}

func TestMapMetric(t *testing.T) {
	resource := MockResource()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}

	emptyList := []*armmonitor.MetricDefinition{}

	emptyMetricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: emptyList,
	}

	metricConfig := azure.MetricConfig{Name: []string{"*"}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}, ServiceType: []string{"blob"}}
	client := azure.NewMockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(emptyMetricDefinitions, nil)
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "no metric definitions were found for resource 123 and namespace Microsoft.Storage/storageAccounts")
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return mapped metrics correctly", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "Microsoft.Storage/storageAccounts")
		assert.Equal(t, metrics[1].ResourceId, "123")
		assert.Equal(t, metrics[1].Namespace, "Microsoft.Storage/storageAccounts")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension(nil))
		assert.Equal(t, metrics[1].Dimensions, []azure.Dimension(nil))

		//order of elements can be different when running the test
		assert.Equal(t, len(metrics), 4)
		for _, metricValue := range metrics {
			assert.Equal(t, metricValue.Aggregations, "Average")
			assert.Equal(t, len(metricValue.Names), 1)
			assert.Contains(t, []string{"TotalRequests", "Capacity"}, metricValue.Names[0])
			if reflect.DeepEqual(metricValue.Names, []string{"Capacity"}) {
				assert.Equal(t, metricValue.TimeGrain, "PT1H")
			} else {
				assert.Equal(t, metricValue.TimeGrain, "PT5M")
			}
		}
		m.AssertExpectations(t)
	})
}

func TestFilterOnTimeGrain(t *testing.T) {
	var list = []armmonitor.MetricDefinition{
		{MetricAvailabilities: availability1},
		{MetricAvailabilities: availability2},
		{MetricAvailabilities: availability3},
	}
	response := groupOnTimeGrain(list)
	assert.Equal(t, len(response), 2)
	result := [][]armmonitor.MetricDefinition{
		{
			{MetricAvailabilities: availability1},
		},
		{
			{MetricAvailabilities: availability2},
			{MetricAvailabilities: availability3},
		},
	}
	for key, availabilities := range response {
		assert.Contains(t, []string{time2, time3}, key)
		assert.Contains(t, result, availabilities)
	}
}

func TestRetrieveSupportedMetricAvailability(t *testing.T) {
	response := retrieveSupportedMetricAvailability(availability1)
	assert.Equal(t, response, time2)
	response = retrieveSupportedMetricAvailability(availability2)
	assert.Equal(t, response, time3)
	response = retrieveSupportedMetricAvailability(availability3)
	assert.Equal(t, response, time3)
}

func TestRetrieveServiceNamespace(t *testing.T) {
	var test = "Microsoft.Storage/storageAccounts/tableServices"
	response := retrieveServiceNamespace(test)
	assert.Equal(t, response, "/tableServices")
	test = "Microsoft.Storage/storageAccounts"
	response = retrieveServiceNamespace(test)
	assert.Equal(t, response, "")
}
