// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	countUnit = armmonitor.MetricUnit("Count")
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

	t.Run("multiple aggregation types", func(t *testing.T) {
		client := NewMockClient()
		referenceTime := time.Now().UTC()
		client.ResourceConfigurations = ResourceConfiguration{
			Metrics: []Metric{
				{
					Namespace:    "Microsoft.EventHub/Namespaces",
					Names:        []string{"ActiveConnections"},
					Aggregations: "Maximum,Minimum,Average",
					TimeGrain:    "PT1M",
				},
			},
		}

		m := &MockService{}
		m.On(
			"GetMetricValues",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(
			[]armmonitor.Metric{{
				ID: to.Ptr("test"),
				Name: &armmonitor.LocalizableString{
					Value:          to.Ptr("ActiveConnections"),
					LocalizedValue: to.Ptr("ActiveConnections"),
				},
				Timeseries: []*armmonitor.TimeSeriesElement{{
					Data: []*armmonitor.MetricValue{{
						Average:   to.Ptr(1.0),
						Maximum:   to.Ptr(2.0),
						Minimum:   to.Ptr(3.0),
						TimeStamp: to.Ptr(time.Now()),
					}},
				}},
				Type:               to.Ptr("Microsoft.Insights/metrics"),
				Unit:               &countUnit,
				DisplayDescription: to.Ptr("Total Active Connections for Microsoft.EventHub."),
				ErrorCode:          to.Ptr("Success"),
			}},
			"PT1M",
			nil,
		)

		client.AzureMonitorService = m
		mr := MockReporterV2{}

		metricValues := client.GetMetricValues(referenceTime, client.ResourceConfigurations.Metrics, &mr)

		require.Equal(t, len(metricValues), 1)
		require.Equal(t, len(metricValues[0].Values), 1)

		assert.Equal(t, *metricValues[0].Values[0].avg, 1.0)
		assert.Equal(t, *metricValues[0].Values[0].max, 2.0)
		assert.Equal(t, *metricValues[0].Values[0].min, 3.0)

		require.Equal(t, len(client.ResourceConfigurations.Metrics[0].Values), 1)

		m.AssertExpectations(t)
	})

	t.Run("single aggregation types", func(t *testing.T) {
		client := NewMockClient()
		referenceTime := time.Now().UTC()
		timestamp := time.Now().UTC()
		client.ResourceConfigurations = ResourceConfiguration{
			Metrics: []Metric{
				{
					Namespace:    "Microsoft.EventHub/Namespaces",
					Names:        []string{"ActiveConnections"},
					Aggregations: "Maximum",
					TimeGrain:    "PT1M",
				}, {
					Namespace:    "Microsoft.EventHub/Namespaces",
					Names:        []string{"ActiveConnections"},
					Aggregations: "Minimum",
					TimeGrain:    "PT1M",
				}, {
					Namespace:    "Microsoft.EventHub/Namespaces",
					Names:        []string{"ActiveConnections"},
					Aggregations: "Average",
					TimeGrain:    "PT1M",
				},
			},
		}

		m := &MockService{}

		x := []struct {
			aggregation string
			data        []*armmonitor.MetricValue
		}{
			{aggregation: "Maximum", data: []*armmonitor.MetricValue{{Maximum: to.Ptr(3.0), TimeStamp: to.Ptr(timestamp)}}},
			{aggregation: "Minimum", data: []*armmonitor.MetricValue{{Minimum: to.Ptr(1.0), TimeStamp: to.Ptr(timestamp)}}},
			{aggregation: "Average", data: []*armmonitor.MetricValue{{Average: to.Ptr(2.0), TimeStamp: to.Ptr(timestamp)}}},
		}

		for _, v := range x {
			m.On(
				"GetMetricValues",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
				v.aggregation,
				mock.Anything,
			).Return(
				[]armmonitor.Metric{{
					ID: to.Ptr("test"),
					Name: &armmonitor.LocalizableString{
						Value:          to.Ptr("ActiveConnections"),
						LocalizedValue: to.Ptr("ActiveConnections"),
					},
					Timeseries: []*armmonitor.TimeSeriesElement{{
						Data: v.data,
					}},
					Type:               to.Ptr("Microsoft.Insights/metrics"),
					Unit:               &countUnit,
					DisplayDescription: to.Ptr("Total Active Connections for Microsoft.EventHub."),
					ErrorCode:          to.Ptr("Success"),
				}},
				"PT1M",
				nil,
			).Once()
		}

		client.AzureMonitorService = m
		mr := MockReporterV2{}

		metricValues := client.GetMetricValues(referenceTime, client.ResourceConfigurations.Metrics, &mr)

		require.Equal(t, 3, len(metricValues))

		require.Equal(t, 1, len(metricValues[0].Values))
		require.Equal(t, 1, len(metricValues[1].Values))
		require.Equal(t, 1, len(metricValues[2].Values))

		require.NotNil(t, metricValues[0].Values[0].max, "max value is nil")
		require.NotNil(t, metricValues[1].Values[0].min, "min value is nil")
		require.NotNil(t, metricValues[2].Values[0].avg, "avg value is nil")

		assert.Equal(t, *metricValues[0].Values[0].max, 3.0)
		assert.Equal(t, *metricValues[1].Values[0].min, 1.0)
		assert.Equal(t, *metricValues[2].Values[0].avg, 2.0)

		m.AssertExpectations(t)
	})
}
