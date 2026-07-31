// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

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
	client := azure.NewMockBatchClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(emptyMetricDefinitions, nil)

		client.AzureMonitorService = m
		client.ResourceConfigurations.MetricDefinitionsChan = make(chan []azure.Metric)
		client.ResourceConfigurations.ErrorChan = make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		concurrentMapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig, &wg)
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)

		assert.EqualError(t, err, "no metric definitions were found for resource 123 and namespace Microsoft.Storage/storageAccounts", "unexpected error from concurrentMapMetrics")
		assert.Empty(t, collectedMetrics, "no metrics should be collected when the definitions lookup fails")
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
		collectedMetrics, err := waitAndCollectConcurrentMapMetrics(client, &wg)

		assert.NoError(t, err)
		assert.Equal(t, "123", collectedMetrics[0].ResourceId)
		assert.Equal(t, "Microsoft.Storage/storageAccounts", collectedMetrics[0].Namespace)
		assert.Equal(t, "123", collectedMetrics[1].ResourceId)
		assert.Equal(t, "Microsoft.Storage/storageAccounts", collectedMetrics[1].Namespace)
		assert.Equal(t, []azure.Dimension(nil), collectedMetrics[0].Dimensions)
		assert.Equal(t, []azure.Dimension(nil), collectedMetrics[1].Dimensions)

		//order of elements can be different when running the test
		assert.Len(t, collectedMetrics, 4)
		for _, metricValue := range collectedMetrics {
			assert.Equal(t, "Average", metricValue.Aggregations)
			assert.Len(t, metricValue.Names, 1)
			assert.Contains(t, []string{"TotalRequests", "Capacity"}, metricValue.Names[0])
			if reflect.DeepEqual(metricValue.Names, []string{"Capacity"}) {
				assert.Equal(t, "PT1H", metricValue.TimeGrain)
			} else {
				assert.Equal(t, "PT5M", metricValue.TimeGrain)
			}
		}
		m.AssertExpectations(t)
	})
}
