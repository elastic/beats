// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

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

	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
	countUnit = armmonitor.Unit("Count")
)

func mockMapResourceMetrics(client *Client, resources []*armresources.GenericResourceExpanded, resourceConfig ResourceConfig) ([]Metric, error) {
	return nil, nil
}

func TestInitResources(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	t.Run("return error when no resource options were configured", func(t *testing.T) {
		client := NewMockClient(logger)
		err := client.InitResources(mockMapResourceMetrics)
		assert.Error(t, err, "no resource options were configured")
	})
	t.Run("return error no resources were found", func(t *testing.T) {
		client := NewMockClient(logger)
		client.Config = resourceQueryConfig
		m := &MockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*armresources.GenericResourceExpanded{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.InitResources(mockMapResourceMetrics)
		assert.Error(t, err, "no resources were found based on all the configurations options entered")
		assert.Empty(t, client.ResourceConfigurations.Metrics)
		m.AssertExpectations(t)
	})
}

func TestGetMetricValues(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	client := NewMockClient(logger)
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
		assert.Empty(t, metrics)
		assert.Empty(t, client.ResourceConfigurations.Metrics[0].Values)
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
		assert.Empty(t, metricValues)
		assert.Empty(t, client.ResourceConfigurations.Metrics[0].Values)
		m.AssertExpectations(t)
	})

	t.Run("multiple aggregation types", func(t *testing.T) {
		client := NewMockClient(logger)
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

		require.Len(t, metricValues, 1)
		require.Len(t, metricValues[0].Values, 1)

		assert.InDelta(t, 1.0, *metricValues[0].Values[0].avg, 0.001)
		assert.InDelta(t, 2.0, *metricValues[0].Values[0].max, 0.001)
		assert.InDelta(t, 3.0, *metricValues[0].Values[0].min, 0.001)

		require.Len(t, client.ResourceConfigurations.Metrics[0].Values, 1)

		m.AssertExpectations(t)
	})

	t.Run("single aggregation types", func(t *testing.T) {
		client := NewMockClient(logger)
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

		require.Len(t, metricValues, 3)

		require.Len(t, metricValues[0].Values, 1)
		require.Len(t, metricValues[1].Values, 1)
		require.Len(t, metricValues[2].Values, 1)

		require.NotNil(t, metricValues[0].Values[0].max, "max value is nil")
		require.NotNil(t, metricValues[1].Values[0].min, "min value is nil")
		require.NotNil(t, metricValues[2].Values[0].avg, "avg value is nil")

		assert.InDelta(t, 3.0, *metricValues[0].Values[0].max, 0.001)
		assert.InDelta(t, 1.0, *metricValues[1].Values[0].min, 0.001)
		assert.InDelta(t, 2.0, *metricValues[2].Values[0].avg, 0.001)

		m.AssertExpectations(t)
	})
}
