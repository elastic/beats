// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package container

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
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
	requird := true
	notRequired := false
	dimension := "status"
	dimensionValue := "*"
	defs := []insights.MetricDefinition{
		{
			Name:                      &insights.LocalizableString{Value: &metric1},
			PrimaryAggregationType:    insights.Average,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Maximum, insights.Count, insights.Total, insights.Average},
			IsDimensionRequired:       &notRequired,
		},
		{
			Name:                      &insights.LocalizableString{Value: &metric2},
			PrimaryAggregationType:    insights.Average,
			SupportedAggregationTypes: &[]insights.AggregationType{insights.Average, insights.Count, insights.Minimum},
			IsDimensionRequired:       &requird,
			Dimensions: &[]insights.LocalizableString{{
				Value:          &dimension,
				LocalizedValue: &dimensionValue,
			}},
		},
	}
	return &defs
}

func TestMapMetric(t *testing.T) {
	resource := MockResource()
	namespace := MockNamespace()
	metricDefinitions := insights.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	emptyList := []insights.MetricDefinition{}
	emptyMetricDefinitions := insights.MetricDefinitionCollection{
		Value: &emptyList,
	}
	metricConfig := azure.MetricConfig{Name: []string{"*"}}
	var resourceConfig = azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient()
	t.Run("return error when the metric namespaces api call returns an error", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricNamespaces", mock.Anything).Return(insights.MetricNamespaceCollection{}, errors.New("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []resources.GenericResource{resource}, resourceConfig)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no metric namespaces were found for resource 123: invalid resource ID")
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricNamespaces", mock.Anything).Return(namespace, nil)
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(emptyMetricDefinitions, nil)
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []resources.GenericResource{resource}, resourceConfig)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no metric definitions were found for resource 123 and namespace namespace.")
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return mapped metrics correctly", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricNamespaces", mock.Anything).Return(namespace, nil)
		m.On("GetMetricDefinitions", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metrics, err := mapMetrics(client, []resources.GenericResource{resource}, resourceConfig)
		assert.Nil(t, err)
		assert.Equal(t, metrics[0].Resource.ID, "123")
		assert.Equal(t, metrics[0].Resource.Name, "resourceName")
		assert.Equal(t, metrics[0].Resource.Type, "resourceType")
		assert.Equal(t, metrics[0].Resource.Location, "resourceLocation")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[1].Resource.ID, "123")
		assert.Equal(t, metrics[1].Resource.Name, "resourceName")
		assert.Equal(t, metrics[1].Resource.Type, "resourceType")
		assert.Equal(t, metrics[1].Resource.Location, "resourceLocation")
		assert.Equal(t, metrics[1].Namespace, "namespace")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension(nil))
		assert.Equal(t, metrics[1].Dimensions, []azure.Dimension(nil))

		//order of elements can be different when running the test
		if metrics[0].Aggregations == "Average" {
			assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity"})
		} else {
			assert.Equal(t, metrics[0].Names, []string{"BytesRead"})
			assert.Equal(t, metrics[0].Aggregations, "Minimum")
		}

		m.AssertExpectations(t)
	})
}
