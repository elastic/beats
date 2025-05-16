// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var count = azmetrics.MetricUnit("Count")

func mockConcurrentMapResourceMetrics(client *BatchClient, resources []*armresources.GenericResourceExpanded, resourceConfig ResourceConfig, wg *sync.WaitGroup) {
}

func TestInitResourcesForBatch(t *testing.T) {
	t.Run("return error when no resource options defined", func(t *testing.T) {
		client := NewMockBatchClient()
		err := client.InitResources(mockConcurrentMapResourceMetrics)
		assert.Error(t, err, "no resource options defined")
	})
	t.Run("return error failed to retrieve resources", func(t *testing.T) {
		client := NewMockBatchClient()
		client.Config = resourceQueryConfig
		m := &MockService{}
		m.On("GetResourceDefinitions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*armresources.GenericResourceExpanded{}, errors.New("invalid resource query"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		err := client.InitResources(mockConcurrentMapResourceMetrics)
		assert.Error(t, err, "failed to retrieve resources: invalid resource query")
		m.AssertExpectations(t)
	})
}

func TestGetMetricsInBatch(t *testing.T) {
	client := NewMockBatchClient()
	client.Config = resourceIDConfig

	t.Run("return no error when no metric values are returned but log and send event", func(t *testing.T) {

		criteria := ResDefGroupingCriteria{
			Namespace:      "namespace",
			SubscriptionID: "subscription",
			Location:       "West Europe",
			Names:          strings.Join([]string{"TotalRequests", "Capacity"}, ","),
			TimeGrain:      "PT1M",
			Dimensions:     getDimensionKey([]Dimension{{Name: "location", Value: "West Europe"}}),
		}
		metrics := []Metric{
			{
				Namespace:    "namespace",
				Names:        []string{"TotalRequests", "Capacity"},
				Aggregations: "Average,Total",
				Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
			},
		}

		groupedMetrics := map[ResDefGroupingCriteria][]Metric{
			criteria: metrics,
		}
		referenceTime := time.Now().UTC().Truncate(time.Second)
		client.ResourceConfigurations = ConcurrentResourceConfig{
			MetricDefinitions: MetricDefinitions{
				Update: true,
				Metrics: map[string][]Metric{
					"resourceId1": {
						{
							Namespace:    "namespace",
							Names:        []string{"TotalRequests", "Capacity"},
							Aggregations: "Average,Total",
							Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
						},
					},
				},
			},
		}
		m := &MockService{}
		m.On("QueryResources", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().
			Return([]azmetrics.MetricData{}, errors.New("invalid parameters or no metrics found"))
		client.AzureMonitorService = m
		mr := MockReporterV2{}
		mr.On("Error", mock.Anything).Return(true)
		results := client.GetMetricsInBatch(groupedMetrics, referenceTime, &mr)
		assert.Equal(t, len(results), 0)
		m.AssertExpectations(t)
	})

	t.Run("multiple aggregation types", func(t *testing.T) {
		client := NewMockBatchClient()
		criteria := ResDefGroupingCriteria{
			Namespace:      "Microsoft.EventHub/Namespaces",
			SubscriptionID: "subscription",
			Location:       "West Europe",
			Names:          strings.Join([]string{"ActiveConnections"}, ","),
			TimeGrain:      "PT1M",
			Dimensions:     getDimensionKey([]Dimension{{Name: "location", Value: "West Europe"}}),
		}
		metricsDef := []Metric{
			{
				Namespace:    "Microsoft.EventHub/Namespaces",
				Names:        []string{"ActiveConnections"},
				Aggregations: "Maximum,Minimum,Average",
				Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
			},
		}

		groupedMetrics := map[ResDefGroupingCriteria][]Metric{
			criteria: metricsDef,
		}
		referenceTime := time.Now().UTC()
		m := &MockService{}
		metrics := []azmetrics.Metric{
			{
				ID: to.Ptr("test"),
				Name: &azmetrics.LocalizableString{
					Value:          to.Ptr("ActiveConnections"),
					LocalizedValue: to.Ptr("Active Connections"),
				},
				TimeSeries: []azmetrics.TimeSeriesElement{
					{
						Data: []azmetrics.MetricValue{
							{
								Average:   to.Ptr(1.0),
								Maximum:   to.Ptr(2.0),
								Minimum:   to.Ptr(3.0),
								TimeStamp: to.Ptr(time.Now()),
							},
						},
					},
				},
				Type:               to.Ptr("Microsoft.Insights/metrics"),
				Unit:               &count,
				DisplayDescription: to.Ptr("Total Active Connections for Microsoft.EventHub."),
				ErrorCode:          to.Ptr("Success"),
			},
		}

		m.On("QueryResources", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			[]azmetrics.MetricData{{
				EndTime:        to.Ptr("2025-01-28T15:00:00Z"),
				StartTime:      to.Ptr("2025-01-28T14:00:00Z"),
				Values:         metrics,
				Interval:       to.Ptr("PT1H"),
				Namespace:      to.Ptr("Microsoft.EventHub/Namespaces"),
				ResourceID:     to.Ptr("resourceId1"),
				ResourceRegion: to.Ptr("West Europe")}},
			nil,
		)

		client.AzureMonitorService = m
		mr := MockReporterV2{}

		metricValues := client.GetMetricsInBatch(groupedMetrics, referenceTime, &mr)
		require.Equal(t, len(metricValues), 1)
		require.Equal(t, len(metricValues[0].Values), 1)

		assert.Equal(t, *metricValues[0].Values[0].avg, 1.0)
		assert.Equal(t, *metricValues[0].Values[0].max, 2.0)
		assert.Equal(t, *metricValues[0].Values[0].min, 3.0)

		m.AssertExpectations(t)
	})
}
