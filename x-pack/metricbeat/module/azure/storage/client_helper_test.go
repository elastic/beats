// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/azure"
)

var (
	time1         = "PT1M"
	time2         = "PT5M"
	time3         = "PT1H"
	availability1 = []insights.MetricAvailability{
		{TimeGrain: &time1},
		{TimeGrain: &time2},
	}
	availability2 = []insights.MetricAvailability{
		{TimeGrain: &time3},
	}
	availability3 = []insights.MetricAvailability{
		{TimeGrain: &time1},
		{TimeGrain: &time3},
	}
)

func MockResource() resources.GenericResourceExpanded {
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

func MockNamespace() insights.MetricNamespaceCollection {
	name := "namespace"
	property := insights.MetricNamespaceName{
		MetricNamespaceName: &name,
	}
	namespace := insights.MetricNamespace{
		Name:       &name,
		Properties: &property,
	}
	list := []insights.MetricNamespace{namespace}
	return insights.MetricNamespaceCollection{
		Value: &list,
	}
}

func MockMetricDefinitions() *[]insights.MetricDefinition {
	metric1 := "TotalRequests"
	metric2 := "Capacity"
	defs := []insights.MetricDefinition{
		{
			Name:                      &insights.LocalizableString{Value: &metric1},
			PrimaryAggregationType:    insights.Average,
			MetricAvailabilities:      &availability1,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Maximum, insights.Count, insights.Total, insights.Average},
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric2},
			PrimaryAggregationType:    insights.Average,
			MetricAvailabilities:      &availability2,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
		},
	}
	return &defs
}

func TestMapMetric(t *testing.T) {
	resource := MockResource()
	metricDefinitions := insights.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	emptyList := []insights.MetricDefinition{}
	emptyMetricDefinitions := insights.MetricDefinitionCollection{
		Value: &emptyList,
	}
	metricConfig := azure.MetricConfig{Name: []string{"*"}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}, ServiceType: []string{"blob"}}
	client := azure.NewMockClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(emptyMetricDefinitions, nil)
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []resources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "no metric definitions were found for resource 123 and namespace Microsoft.Storage/storageAccounts.")
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return mapped metrics correctly", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metrics, err := mapMetrics(client, []resources.GenericResourceExpanded{resource}, resourceConfig)
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
	var list = []insights.MetricDefinition{
		{MetricAvailabilities: &availability1},
		{MetricAvailabilities: &availability2},
		{MetricAvailabilities: &availability3},
	}
	response := groupOnTimeGrain(list)
	assert.Equal(t, len(response), 2)
	result := [][]insights.MetricDefinition{
		{
			{MetricAvailabilities: &availability1},
		},
		{
			{MetricAvailabilities: &availability2},
			{MetricAvailabilities: &availability3},
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
