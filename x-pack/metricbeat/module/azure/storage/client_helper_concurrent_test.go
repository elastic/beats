// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package storage

import (
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

func TestConcurrentMapMetrics(t *testing.T) {
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
	client := azure.NewMockBatchClient()
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(emptyMetricDefinitions, nil)

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
		assert.Equal(t, error.Error(), "no metric definitions were found for resource 123 and namespace Microsoft.Storage/storageAccounts")
		assert.Equal(t, collectedMetrics, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return mapped metrics correctly", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
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

		assert.NoError(t, error)
		assert.Equal(t, collectedMetrics[0].ResourceId, "123")
		assert.Equal(t, collectedMetrics[0].Namespace, "Microsoft.Storage/storageAccounts")
		assert.Equal(t, collectedMetrics[1].ResourceId, "123")
		assert.Equal(t, collectedMetrics[1].Namespace, "Microsoft.Storage/storageAccounts")
		assert.Equal(t, collectedMetrics[0].Dimensions, []azure.Dimension(nil))
		assert.Equal(t, collectedMetrics[1].Dimensions, []azure.Dimension(nil))

		//order of elements can be different when running the test
		assert.Equal(t, len(collectedMetrics), 4)
		for _, metricValue := range collectedMetrics {
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
