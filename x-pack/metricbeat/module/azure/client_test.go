// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	resourceIDConfig = Config{
		Resources: []ResourceConfig{
			{Id: []string{"123"},
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

func mockMapResourceMetrics(client *Client, resources []*armresources.GenericResourceExpanded, resourceConfig ResourceConfig) ([]Metric, error) {
	return nil, nil
}

func TestInitResources(t *testing.T) {
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := NewMockClient()
		err := client.InitResources(mockMapResourceMetrics)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client := NewMockClient()
		client.Config = resourceQueryConfig
		m := &MockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*armresources.GenericResourceExpanded{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.InitResources(mockMapResourceMetrics)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Equal(t, len(client.ResourceConfigurations.Metrics), 0)
		m.AssertExpectations(t)
	})
}

func TestGetMetricValues(t *testing.T) {
	client := NewMockClient()
	client.Config = resourceIDConfig

	t.Run("return no error when no metric values are returned but log and send event", func(t *testing.T) {
		referenceTime := time.Now().UTC().Truncate(time.Second)
		client.ResourceConfigurations = ResourceConfiguration{
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
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().
			Return([]armmonitor.Metric{}, "", errors.New("invalid parameters or no metrics found"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		metrics := client.GetMetricValues(referenceTime, client.ResourceConfigurations.Metrics, &mr)
		assert.Equal(t, len(metrics), 0)
		assert.Equal(t, len(client.ResourceConfigurations.Metrics[0].Values), 0)
		m.AssertExpectations(t)
	})
	t.Run("return metric values", func(t *testing.T) {
		referenceTime := time.Now().UTC().Truncate(time.Second)
		client.ResourceConfigurations = ResourceConfiguration{
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
		m.On("GetMetricValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return([]armmonitor.Metric{}, "", errors.New("invalid parameters or no metrics found"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		metricValues := client.GetMetricValues(referenceTime, client.ResourceConfigurations.Metrics, &mr)
		assert.Equal(t, len(metricValues), 0)
		assert.Equal(t, len(client.ResourceConfigurations.Metrics[0].Values), 0)
		m.AssertExpectations(t)
	})
}
