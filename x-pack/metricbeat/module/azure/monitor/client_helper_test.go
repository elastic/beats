// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var (
	oneMinuteDuration    = `PT1M`
	thirtyMinuteDuration = `PT30M`
	oneHrDuration        = `PT1H`
	sixHrDuration        = `PT6H`
)

func MockResourceExpanded() *armresources.GenericResourceExpanded {
	id := "123"
	name := "resourceName"
	location := "resourceLocation"
	rType := "resourceType"

	return &armresources.GenericResourceExpanded{
		ID:       &id,
		Name:     &name,
		Location: &location,
		Type:     &rType,
	}
}

func MockMetricDefinitions() []*armmonitor.MetricDefinition {
	var (
		metric1 = "TotalRequests"
		metric2 = "Capacity"
		metric3 = "BytesRead"

		aggregationTypeAverage = armmonitor.AggregationTypeAverage
		aggregationTypeCount   = armmonitor.AggregationTypeCount
		aggregationTypeMinimum = armmonitor.AggregationTypeMinimum
		aggregationTypeMaximum = armmonitor.AggregationTypeMaximum
		aggregationTypeTotal   = armmonitor.AggregationTypeTotal
	)

	defs := []*armmonitor.MetricDefinition{
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric1},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeMaximum,
				&aggregationTypeCount,
				&aggregationTypeTotal,
				&aggregationTypeAverage,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &thirtyMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			}, // TODO: pick up here and add to other defs as well
		},
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric2},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeAverage,
				&aggregationTypeCount,
				&aggregationTypeMinimum,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &oneMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			},
		},
		{
			Name:                   &armmonitor.LocalizableString{Value: &metric3},
			PrimaryAggregationType: &aggregationTypeAverage,
			SupportedAggregationTypes: []*armmonitor.AggregationType{
				&aggregationTypeAverage,
				&aggregationTypeCount,
				&aggregationTypeMinimum,
			},
			MetricAvailabilities: []*armmonitor.MetricAvailability{
				{TimeGrain: &thirtyMinuteDuration},
				{TimeGrain: &oneHrDuration},
				{TimeGrain: &sixHrDuration},
			},
		},
	}
	return defs
}

func TestMapMetricWithConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace",
		Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}},
		Timegrain:  oneHrDuration}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity", "BytesRead"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		assert.Equal(t, metrics[0].TimeGrain, oneHrDuration)
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when specific metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"TotalRequests", "Capacity"}
		metricConfig.Aggregations = []string{"Average"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		assert.True(t, len(metrics) > 0)
		assert.Equal(t, metrics[0].ResourceId, "123")
		assert.Equal(t, metrics[0].Namespace, "namespace")
		assert.Equal(t, metrics[0].Names, []string{"TotalRequests", "Capacity"})
		assert.Equal(t, metrics[0].Aggregations, "Average")
		assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
		assert.Equal(t, metrics[0].TimeGrain, oneHrDuration)
		m.AssertExpectations(t)
	})
}

func TestInvalidConfiguredTimegrain(t *testing.T) {
	// expected behavior is to simply skip the metrics that
	// are not compatible with the configured timegrain
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace",
		Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}},
		// one-minute timegrain is not supported by some metrics
		Timegrain: oneMinuteDuration}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))

	m := &azure.MockService{}
	m.On("GetMetricDefinitionsWithRetry", mock.Anything,
		mock.Anything).Return(metricDefinitions, nil)
	client.AzureMonitorService = m
	metricConfig.Name = []string{"*"}
	resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
	metrics, err := mapMetrics(client,
		[]*armresources.GenericResourceExpanded{resource}, resourceConfig)

	assert.NoError(t, err)

	assert.Len(t, metrics, 1)
	assert.Equal(t, metrics[0].ResourceId, "123")
	assert.Equal(t, metrics[0].Namespace, "namespace")
	assert.Equal(t, metrics[0].Names, []string{"Capacity"})
	assert.Equal(t, metrics[0].Aggregations, "Average")
	assert.Equal(t, metrics[0].Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
	assert.Equal(t, metrics[0].TimeGrain, oneMinuteDuration)
	m.AssertExpectations(t)
}

func TestMapMetricNoConfiguredTimegrain(t *testing.T) {
	resource := MockResourceExpanded()
	metricDefinitions := armmonitor.MetricDefinitionCollection{
		Value: MockMetricDefinitions(),
	}
	metricConfig := azure.MetricConfig{Namespace: "namespace", Dimensions: []azure.DimensionConfig{{Name: "location", Value: "West Europe"}}}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	t.Run("return error when no metric definitions were found", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(armmonitor.MetricDefinitionCollection{}, fmt.Errorf("invalid resource ID"))
		client.AzureMonitorService = m
		metric, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.Error(t, err)
		assert.Equal(t, metric, []azure.Metric(nil))
		m.AssertExpectations(t)
	})
	t.Run("return all metrics when all metric names and aggregations were configured", func(t *testing.T) {
		m := &azure.MockService{}
		m.On("GetMetricDefinitionsWithRetry", mock.Anything, mock.Anything).Return(metricDefinitions, nil)
		client.AzureMonitorService = m
		metricConfig.Name = []string{"*"}
		resourceConfig.Metrics = []azure.MetricConfig{metricConfig}
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		// we should have two groups, one per first timegrain value
		assert.Len(t, metrics, 2)
		// this for loop with the switch statement is necessary because the ordering of timegrains is non-deterministic
		// due to map iteration. Without a configured timegrain, we are iterating over a map
		for _, metric := range metrics {
			switch metric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"Capacity"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			case thirtyMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"TotalRequests", "BytesRead"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			default:
				// cannot have any other cases
				t.FailNow()
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
		metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, resourceConfig)
		assert.NoError(t, err)

		assert.True(t, len(metrics) > 0)

		// we should have two groups, one per first timegrain value
		assert.Len(t, metrics, 2)
		// this for loop with the switch statement is necessary because the ordering of timegrains is non-deterministic
		// due to map iteration. Without a configured timegrain, we are iterating over a map
		for _, metric := range metrics {
			switch metric.TimeGrain {
			case oneMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"Capacity"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			case thirtyMinuteDuration:
				assert.Equal(t, metric.ResourceId, "123")
				assert.Equal(t, metric.Namespace, "namespace")
				assert.Equal(t, metric.Names, []string{"TotalRequests"})
				assert.Equal(t, metric.Aggregations, "Average")
				assert.Equal(t, metric.Dimensions, []azure.Dimension{{Name: "location", Value: "West Europe"}})
			default:
				// cannot have any other cases
				t.FailNow()
			}
		}

		m.AssertExpectations(t)
	})
}

func TestFilterSConfiguredMetrics(t *testing.T) {
	selectedRange := []string{"TotalRequests", "Capacity", "CPUUsage"}
	intersection, difference := filterConfiguredMetrics(selectedRange, MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"TotalRequests", "Capacity"})
	assert.Equal(t, difference, []string{"CPUUsage"})
}

func TestFilterAggregations(t *testing.T) {
	selectedRange := []string{"Average", "Minimum"}
	intersection, difference := filterAggregations(selectedRange, MockMetricDefinitions())
	assert.Equal(t, intersection, []string{"Average"})
	assert.Equal(t, difference, []string{"Minimum"})
}

func TestFilter(t *testing.T) {
	str := []string{"hello", "test", "goodbye", "test"}
	filtered := filter(str)
	assert.Equal(t, len(filtered), 3)
}

func TestIntersections(t *testing.T) {
	firstStr := []string{"test1", "test2", "test2", "test3"}
	sercondStr := []string{"test4", "test5", "test2", "test5", "test3"}
	intersection, difference := intersections(firstStr, sercondStr)
	assert.Equal(t, intersection, []string{"test2", "test3"})
	assert.Equal(t, difference, []string{"test4", "test5"})

	firstStr = []string{"test1", "test2", "test2", "test3"}
	sercondStr = []string{"test4", "test5", "test5"}
	intersection, difference = intersections(firstStr, sercondStr)
	assert.Equal(t, len(intersection), 0)
	assert.Equal(t, difference, []string{"test4", "test5"})
}

func TestGetMetricDefinitionsByNames(t *testing.T) {
	metrics := []string{"TotalRequests", "CPUUsage"}
	result := getMetricDefinitionsByNames(MockMetricDefinitions(), metrics)
	assert.Equal(t, len(result), 1)
	assert.Equal(t, *result[0].Name.Value, "TotalRequests")
}

func TestMapMetricsSkipsUnsupportedPlatformMetricNamespace(t *testing.T) {
	unsupported := resourceExpanded("activity-log-alert")
	supported := resourceExpanded("key-vault")
	metricDefinitions := armmonitor.MetricDefinitionCollection{Value: MockMetricDefinitions()}
	metricConfig := azure.MetricConfig{
		Name:              []string{"*"},
		Namespace:         "Microsoft.Insights/activityLogAlerts",
		Timegrain:         oneHrDuration,
		IgnoreUnsupported: true,
	}
	resourceConfig := azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	service := &azure.MockService{}
	service.On("GetMetricDefinitionsWithRetry", "activity-log-alert", metricConfig.Namespace).
		Return(armmonitor.MetricDefinitionCollection{}, unsupportedPlatformMetricNamespaceError(metricConfig.Namespace)).Once()
	service.On("GetMetricDefinitionsWithRetry", "key-vault", metricConfig.Namespace).
		Return(metricDefinitions, nil).Once()
	client.AzureMonitorService = service

	metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{unsupported, supported}, resourceConfig)

	assert.NoError(t, err, "HTTP 400 for an unsupported platform metric namespace should be skipped when ignore_unsupported is true")
	assert.NotEmpty(t, metrics, "metrics for the supported resource should still be collected")
	for _, metric := range metrics {
		assert.Equal(t, "key-vault", metric.ResourceId, "only the supported resource should be mapped")
	}
	service.AssertExpectations(t)
}

func resourceExpanded(id string) *armresources.GenericResourceExpanded {
	name := "resourceName"
	location := "resourceLocation"
	resourceType := "resourceType"
	return &armresources.GenericResourceExpanded{
		ID:       &id,
		Name:     &name,
		Location: &location,
		Type:     &resourceType,
	}
}

func unsupportedPlatformMetricNamespaceError(namespace string) error {
	return azureBadRequest(fmt.Sprintf("%s is not a supported platform metric namespace", namespace))
}

func azureBadRequest(message string) error {
	body := fmt.Sprintf(`{"code":"BadRequest","message":"%s"}`, message)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://management.azure.com/metricDefinitions", nil)
	if err != nil {
		panic(err)
	}
	resp := &http.Response{
		Status:        "400 Bad Request",
		StatusCode:    http.StatusBadRequest,
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        make(http.Header),
		Request:       req,
		ContentLength: int64(len(body)),
	}
	return runtime.NewResponseError(resp)
}

func TestMapMetricsUnsupportedNamespaceStaysFatalWithoutIgnoreUnsupported(t *testing.T) {
	resource := resourceExpanded("activity-log-alert")
	metricConfig := azure.MetricConfig{
		Name:      []string{"*"},
		Namespace: "Microsoft.Insights/activityLogAlerts",
		Timegrain: oneHrDuration,
	}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	service := &azure.MockService{}
	service.On("GetMetricDefinitionsWithRetry", "activity-log-alert", metricConfig.Namespace).
		Return(armmonitor.MetricDefinitionCollection{}, unsupportedPlatformMetricNamespaceError(metricConfig.Namespace)).Once()
	client.AzureMonitorService = service

	metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}})

	assert.Error(t, err, "unsupported namespace HTTP 400 should still fail when ignore_unsupported is false")
	assert.Empty(t, metrics, "no metrics should be returned when mapping fails")
	service.AssertExpectations(t)
}

func TestMapMetricsDoesNotSkipOtherDefinitionErrors(t *testing.T) {
	resource := resourceExpanded("activity-log-alert")
	metricConfig := azure.MetricConfig{
		Name:              []string{"*"},
		Namespace:         "Microsoft.Insights/activityLogAlerts",
		Timegrain:         oneHrDuration,
		IgnoreUnsupported: true,
	}
	client := azure.NewMockClient(logptest.NewTestingLogger(t, ""))
	service := &azure.MockService{}
	service.On("GetMetricDefinitionsWithRetry", "activity-log-alert", metricConfig.Namespace).
		Return(armmonitor.MetricDefinitionCollection{}, azureBadRequest("The resource id is invalid")).Once()
	client.AzureMonitorService = service

	metrics, err := mapMetrics(client, []*armresources.GenericResourceExpanded{resource}, azure.ResourceConfig{Metrics: []azure.MetricConfig{metricConfig}})

	assert.Error(t, err, "ignore_unsupported should not skip a 400 that is not an unsupported platform metric namespace")
	assert.Empty(t, metrics, "no metrics should be returned when mapping fails")
	service.AssertExpectations(t)
}
