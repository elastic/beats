// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	resourceIDConfig = Config{
		Resources: []ResourceConfig{
			{ID: []string{"123"},
				Metrics: []MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
	resourceQueryConfig = Config{
		Resources: []ResourceConfig{
			{
				Query: "query",
				Metrics: []MetricConfig{
					{
						Name: []string{"hello", "test"},
					},
				}}},
	}
)

func mockMapMetric(client *Client, metric MetricConfig, resource resources.GenericResource) ([]Metric, error) {
	return nil, nil
}

func TestInitResources(t *testing.T) {
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := NewMockClient()
		mr := MockReporterV2{}
		err := client.InitResources(mockMapMetric, &mr)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client := NewMockClient()
		client.Config = resourceQueryConfig
		m := &MockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(resources.ListResultPage{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.InitResources(mockMapMetric, &mr)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Equal(t, len(client.Resources.Metrics), 0)
		m.AssertExpectations(t)
	})
}

func TestGetMetricValues(t *testing.T) {
	client := NewMockClient()
	client.Config = resourceIDConfig
	t.Run("return no error when no metric values are returned but log and send event", func(t *testing.T) {
		client.Resources = ResourceConfiguration{
			Metrics: []Metric{
				{
					Namespace:    "namespace",
					Names:        []string{"TotalRequests,Capacity"},
					Aggregations: "Average,Total",
					Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
				},
			},
		}
		m := &MockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]insights.Metric{}, errors.New("invalid parameters or no metrics found"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.GetMetricValues(&mr)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Resources.Metrics[0].Values), 0)
		m.AssertExpectations(t)
	})
	t.Run("return metric values", func(t *testing.T) {
		client.Resources = ResourceConfiguration{
			Metrics: []Metric{
				{
					Namespace:    "namespace",
					Names:        []string{"TotalRequests,Capacity"},
					Aggregations: "Average,Total",
					Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
				},
			},
		}
		m := &MockService{}
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]insights.Metric{}, errors.New("invalid parameters or no metrics found"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.GetMetricValues(&mr)
		assert.Nil(t, err)
		assert.Equal(t, len(client.Resources.Metrics[0].Values), 0)
		m.AssertExpectations(t)
	})
}
