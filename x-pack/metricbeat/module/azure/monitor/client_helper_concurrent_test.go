// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package monitor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/mock"

	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func waitAndCollectConcurrentMapMetrics(client *azure.BatchClient, wg *sync.WaitGroup) ([]azure.Metric, error) {
	metricsCh := client.ResourceConfigurations.MetricDefinitionsChan
	errorCh := client.ResourceConfigurations.ErrorChan

	go func() {
		wg.Wait()
		client.Log.Infof("All collections finished. Closing channels ")
		close(metricsCh)
		close(errorCh)
	}()

	var collectedMetrics []azure.Metric
	var collectedErr error
	metricsOpen := true
	errorOpen := true
	for metricsOpen || errorOpen {
		select {
		case resMetricDefinition, ok := <-metricsCh:
			if !ok {
				metricsOpen = false
			} else {
				collectedMetrics = append(collectedMetrics, resMetricDefinition...)
			}
		case err, ok := <-errorCh:
			if ok && err != nil {
				collectedErr = err
			}
			errorOpen = false
		}
	}
	return collectedMetrics, collectedErr
}

func TestConcurrentMapMetricsWithConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace",
		Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}},
		Timegrain:  oneHrDuration}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockBatchClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)
		assert.EqualError(t, err, "invalid resource ID", "unexpected error from concurrentMapMetrics")
		assert.Equal(t, collectedMetrics, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)

		assert.NoError(t, err)
		assert.Equal(t, "123", collectedMetrics[0].ResourceId)
		assert.Equal(t, "namespace", collectedMetrics[0].Namespace)
		assert.Equal(t, []string{"TotalRequests", "Capacity", "BytesRead"}, collectedMetrics[0].Names)
		assert.Equal(t, "Average", collectedMetrics[0].Aggregations)
		assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetrics[0].Dimensions)
		assert.Equal(t, oneHrDuration, collectedMetrics[0].TimeGrain)
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "Capacity"}
		metricConfig.Aggregations = []string{"Average"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)
		assert.NoError(t, err)

		assert.NotEmpty(t, collectedMetrics)
		assert.Equal(t, "123", collectedMetrics[0].ResourceId)
		assert.Equal(t, "namespace", collectedMetrics[0].Namespace)
		assert.Equal(t, []string{"TotalRequests", "Capacity"}, collectedMetrics[0].Names)
		assert.Equal(t, "Average", collectedMetrics[0].Aggregations)
		assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetrics[0].Dimensions)
		assert.Equal(t, oneHrDuration, collectedMetrics[0].TimeGrain)
		m.AssertExpectations(t)
	})
}

func TestConcurrentMapMetricsNoConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace",
		Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockBatchClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)
		assert.EqualError(t, err, "invalid resource ID", "unexpected error from concurrentMapMetrics")
		assert.Equal(t, collectedMetrics, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)

		assert.NoError(t, err)

		// we should have two groups, one per first timegrain value
		assert.Len(t, collectedMetrics, 2)
		// this for loop with the switch statement is necessary because the ordering of timegrains is non-deterministic
		// due to map iteration. Without a configured timegrain, we are iterating over a map
		for _, collectedMetric := range collectedMetrics {
			switch collectedMetric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, "123", collectedMetric.ResourceId)
				assert.Equal(t, "namespace", collectedMetric.Namespace)
				assert.Equal(t, []string{"Capacity"}, collectedMetric.Names)
				assert.Equal(t, "Average", collectedMetric.Aggregations)
				assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetric.Dimensions)
			case thirtyMinuteDuration:
				assert.Equal(t, "123", collectedMetric.ResourceId)
				assert.Equal(t, "namespace", collectedMetric.Namespace)
				assert.Equal(t, []string{"TotalRequests", "BytesRead"}, collectedMetric.Names)
				assert.Equal(t, "Average", collectedMetric.Aggregations)
				assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetric.Dimensions)
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
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)
		assert.NoError(t, err)

		assert.NotEmpty(t, collectedMetrics)

		// we should have two groups, one per first timegrain value
		assert.Len(t, collectedMetrics, 2)
		// this for loop with the switch statement is necessary because the ordering of timegrains is non-deterministic
		// due to map iteration. Without a configured timegrain, we are iterating over a map
		for _, collectedMetric := range collectedMetrics {
			switch collectedMetric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, "123", collectedMetric.ResourceId)
				assert.Equal(t, "namespace", collectedMetric.Namespace)
				assert.Equal(t, []string{"Capacity"}, collectedMetric.Names)
				assert.Equal(t, "Average", collectedMetric.Aggregations)
				assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetric.Dimensions)
			case thirtyMinuteDuration:
				assert.Equal(t, "123", collectedMetric.ResourceId)
				assert.Equal(t, "namespace", collectedMetric.Namespace)
				assert.Equal(t, []string{"TotalRequests"}, collectedMetric.Names)
				assert.Equal(t, "Average", collectedMetric.Aggregations)
				assert.Equal(t, []azure.Dimension{{Name: "location", Value: "West Europe"}}, collectedMetric.Dimensions)
			}
		}

		m.AssertExpectations(t)
	})
}
