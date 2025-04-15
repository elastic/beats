// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
)

func TestConcurrentMapMetrics(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockBatchClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		go func() {
			wg.Wait() // Wait for all the resource collection goroutines to finish
			// Once all the goroutines are done, close the channels
			client.Log.Infof("All collections finished. Closing channels ")
			close(client.ResourceConfigurations.MetricDefinitionsChan)
			close(client.ResourceConfigurations.ErrorChan)
		}()
		var collectedMetrics []azure.Metric
		var error error
		for {
			select {
			case resMetricDefinition, ok := <-client.ResourceConfigurations.MetricDefinitionsChan:
				if !ok {
					client.ResourceConfigurations.MetricDefinitionsChan = nil
				} else {
					collectedMetrics = append(collectedMetrics, resMetricDefinition...)
				}
			case err, ok := <-client.ResourceConfigurations.ErrorChan:
				if ok && err != nil {
					// Handle error received from error channel
					error = err
				}
				// Error channel is closed, stop error handling
				client.ResourceConfigurations.ErrorChan = nil
			}

			// Break the loop when both Data and Error channels are closed
			if client.ResourceConfigurations.MetricDefinitionsChan == nil && client.ResourceConfigurations.ErrorChan == nil {
				break
			}
		}
		assert.Error(t, error)
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
		go func() {
			wg.Wait() // Wait for all the resource collection goroutines to finish
			// Once all the goroutines are done, close the channels
			client.Log.Infof("All collections finished. Closing channels ")
			close(client.ResourceConfigurations.MetricDefinitionsChan)
			close(client.ResourceConfigurations.ErrorChan)
		}()
		var collectedMetrics []azure.Metric
		var error error
		for {
			select {
			case resMetricDefinition, ok := <-client.ResourceConfigurations.MetricDefinitionsChan:
				if !ok {
					client.ResourceConfigurations.MetricDefinitionsChan = nil
				} else {
					collectedMetrics = append(collectedMetrics, resMetricDefinition...)
				}
			case err, ok := <-client.ResourceConfigurations.ErrorChan:
				if ok && err != nil {
					// Handle error received from error channel
					error = err
				}
				// Error channel is closed, stop error handling
				client.ResourceConfigurations.ErrorChan = nil
			}

			// Break the loop when both Data and Error channels are closed
			if client.ResourceConfigurations.MetricDefinitionsChan == nil && client.ResourceConfigurations.ErrorChan == nil {
				break
			}
		}

		assert.NoError(t, error)
		assert.Equal(t, collectedMetrics[0].ResourceId, "123")
		assert.Equal(t, collectedMetrics[0].Namespace, "namespace")
		assert.Equal(t, collectedMetrics[0].Names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, collectedMetrics[0].Aggregations, "Average")
		assert.Equal(t, collectedMetrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
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
		go func() {
			wg.Wait() // Wait for all the resource collection goroutines to finish
			// Once all the goroutines are done, close the channels
			client.Log.Infof("All collections finished. Closing channels ")
			close(client.ResourceConfigurations.MetricDefinitionsChan)
			close(client.ResourceConfigurations.ErrorChan)
		}()
		var collectedMetrics []azure.Metric
		var error error
		for {
			select {
			case resMetricDefinition, ok := <-client.ResourceConfigurations.MetricDefinitionsChan:
				if !ok {
					client.ResourceConfigurations.MetricDefinitionsChan = nil
				} else {
					collectedMetrics = append(collectedMetrics, resMetricDefinition...)
				}
			case err, ok := <-client.ResourceConfigurations.ErrorChan:
				if ok && err != nil {
					// Handle error received from error channel
					error = err
				}
				// Error channel is closed, stop error handling
				client.ResourceConfigurations.ErrorChan = nil
			}

			// Break the loop when both Data and Error channels are closed
			if client.ResourceConfigurations.MetricDefinitionsChan == nil && client.ResourceConfigurations.ErrorChan == nil {
				break
			}
		}
		assert.NoError(t, error)

		assert.True(t, len(collectedMetrics) > 0)
		assert.Equal(t, collectedMetrics[0].ResourceId, "123")
		assert.Equal(t, collectedMetrics[0].Namespace, "namespace")
		assert.Equal(t, collectedMetrics[0].Names, []string{"TotalRequests", "Capacity"})
		assert.Equal(t, collectedMetrics[0].Aggregations, "Average")
		assert.Equal(t, collectedMetrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		m.AssertExpectations(t)
	})
}
