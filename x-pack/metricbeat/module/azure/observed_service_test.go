// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"errors"
	"net/url"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

func TestObservedServiceRecordsGetMetricValues_Success(t *testing.T) {
	reg := monitoring.NewRegistry()
	apiMetrics := newAzureAPIMetrics(reg, logptest.NewTestingLogger(t, ""))
	next := &recordingService{
		getMetricValuesResponse:  []armmonitor.Metric{{}},
		getMetricValuesTimegrain: "PT1M",
	}
	service := newObservedService(next, apiMetrics)

	resp, timegrain, err := service.GetMetricValues(
		"resource",
		"namespace",
		"PT1M",
		"2026-05-07T20:00:00Z/2026-05-07T20:01:00Z",
		[]string{"Requests"},
		"Total",
		"",
	)

	require.NoError(t, err, "GetMetricValues should return the wrapped service result")
	assert.Len(t, resp, 1, "GetMetricValues should return the wrapped response")
	assert.Equal(t, "PT1M", timegrain, "GetMetricValues should return the wrapped timegrain")
	assert.Equal(t, 1, next.getMetricValuesCalls, "GetMetricValues should delegate exactly once")
	assert.Equal(t, uint64(1), metricCount(t, reg, operationGetMetricValues, apiResultSuccess, apiErrorKindNone), "success count should increment")
	assert.Equal(t, int64(1), metricDurationCount(t, reg, operationGetMetricValues, apiResultSuccess, apiErrorKindNone), "success duration should be recorded")
	assert.NotNil(t, reg.Get("api_calls_get_metric_values_success_total"), "metric name must use underscore naming")
}

func TestObservedServiceRecordsQueryResources_Failure(t *testing.T) {
	reg := monitoring.NewRegistry()
	apiMetrics := newAzureAPIMetrics(reg, logptest.NewTestingLogger(t, ""))
	expectedErr := errors.New("query failed")
	next := &recordingService{queryResourcesErr: expectedErr}
	service := newObservedService(next, apiMetrics)

	resp, err := service.QueryResources(
		[]string{"resource"},
		"subscription",
		"namespace",
		"PT1M",
		"2026-05-07T20:00:00Z",
		"2026-05-07T20:01:00Z",
		[]string{"Requests"},
		"Total",
		"",
		"westeurope",
	)

	require.ErrorIs(t, err, expectedErr, "QueryResources should return the wrapped service error")
	assert.Nil(t, resp, "QueryResources should return the wrapped response")
	assert.Equal(t, 1, next.queryResourcesCalls, "QueryResources should delegate exactly once")
	assert.Equal(t, uint64(1), metricCount(t, reg, operationQueryResources, apiResultFailure, apiErrorKindOther), "failure count should increment")
	assert.Equal(t, int64(1), metricDurationCount(t, reg, operationQueryResources, apiResultFailure, apiErrorKindOther), "failure duration should be recorded")
}

func TestObservedServiceClassifiesThrottleFailures_Throttle(t *testing.T) {
	reg := monitoring.NewRegistry()
	apiMetrics := newAzureAPIMetrics(reg, logptest.NewTestingLogger(t, ""))
	next := &recordingService{
		getMetricDefinitionsErr: &azcore.ResponseError{StatusCode: 429},
	}
	service := newObservedService(next, apiMetrics)

	_, err := service.GetMetricDefinitionsWithRetry("resource", "namespace")

	require.Error(t, err, "GetMetricDefinitionsWithRetry should return the wrapped service error")
	assert.Equal(t, uint64(1), metricCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindThrottle), "throttle failure count should increment")
	assert.Equal(t, int64(1), metricDurationCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindThrottle), "throttle duration should be recorded")
}

func TestObservedServiceClassifiesTransportFailures_Transport(t *testing.T) {
	reg := monitoring.NewRegistry()
	apiMetrics := newAzureAPIMetrics(reg, logptest.NewTestingLogger(t, ""))
	next := &recordingService{
		getMetricDefinitionsErr: &url.Error{
			Op:  "Get",
			URL: "https://management.azure.com",
			Err: errors.New("connection refused"),
		},
	}
	service := newObservedService(next, apiMetrics)

	_, err := service.GetMetricDefinitionsWithRetry("resource", "namespace")

	require.Error(t, err, "GetMetricDefinitionsWithRetry should return the wrapped service error")
	assert.Equal(t, uint64(1), metricCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindTransport), "transport failure count should increment")
	assert.Equal(t, int64(1), metricDurationCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindTransport), "transport duration should be recorded")
}

func TestObservedServiceClassifiesAzureResponseErrors_Non429(t *testing.T) {
	reg := monitoring.NewRegistry()
	apiMetrics := newAzureAPIMetrics(reg, logptest.NewTestingLogger(t, ""))
	next := &recordingService{
		getMetricDefinitionsErr: &azcore.ResponseError{StatusCode: 403},
	}
	service := newObservedService(next, apiMetrics)

	_, err := service.GetMetricDefinitionsWithRetry("resource", "namespace")

	require.Error(t, err, "GetMetricDefinitionsWithRetry should return the wrapped service error")
	assert.Equal(t, uint64(1), metricCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindAzureResponse), "non-429 Azure response failure count should increment")
	assert.Equal(t, int64(1), metricDurationCount(t, reg, operationGetMetricDefinitions, apiResultFailure, apiErrorKindAzureResponse), "non-429 Azure response duration should be recorded")
}

func TestNewClientWiresObservedService(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("client wraps service when registry is provided", func(t *testing.T) {
		reg := monitoring.NewRegistry()
		mockService := new(MockService)

		// Simulate the wiring that NewMetricSet → NewClient does:
		// newObservedAzureMonitorService wraps the service with metrics.
		observed := newObservedAzureMonitorService(mockService, reg, logger)

		_, ok := observed.(*observedService)
		assert.True(t, ok, "newObservedAzureMonitorService should return an *observedService when a registry is provided")
	})

	t.Run("client returns original service when no registry is provided", func(t *testing.T) {
		mockService := new(MockService)

		observed := newObservedAzureMonitorService(mockService, nil, logger)

		// When no registry is provided, newObservedAzureMonitorService
		// returns the original service unwrapped (via newObservedService's
		// nil-metrics fast path).
		assert.Equal(t, mockService, observed, "newObservedAzureMonitorService should return the original service when no registry is provided")
	})

	t.Run("batch client wraps service when registry is provided", func(t *testing.T) {
		reg := monitoring.NewRegistry()
		mockService := new(MockService)

		// BatchClient uses the same newObservedAzureMonitorService function.
		observed := newObservedAzureMonitorService(mockService, reg, logger)

		_, ok := observed.(*observedService)
		assert.True(t, ok, "newObservedAzureMonitorService should return an *observedService for batch clients when a registry is provided")
	})
}

func metricCount(t *testing.T, reg *monitoring.Registry, operation apiOperation, result apiResult, errorKind apiErrorKind) uint64 {
	t.Helper()

	name := apiMetricName(operation, result, errorKind, "total")
	counter, ok := reg.Get(name).(*monitoring.Uint)
	require.True(t, ok, "counter metric %s should exist", name)
	return counter.Get()
}

func metricDurationCount(t *testing.T, reg *monitoring.Registry, operation apiOperation, result apiResult, errorKind apiErrorKind) int64 {
	t.Helper()

	name := apiMetricName(operation, result, errorKind, "duration")
	histogram, ok := adapter.GetGoMetrics(reg, name, logptest.NewTestingLogger(t, ""), adapter.Accept).
		Get("histogram").(interface{ Count() int64 })
	require.True(t, ok, "duration histogram %s should exist in monitoring registry", name)
	return histogram.Count()
}

type recordingService struct {
	getResourceDefinitionByIDCalls    int
	getResourceDefinitionByIDResponse armresources.GenericResource
	getResourceDefinitionByIDErr      error

	getResourceDefinitionsCalls    int
	getResourceDefinitionsResponse []*armresources.GenericResourceExpanded
	getResourceDefinitionsErr      error

	getMetricDefinitionsCalls    int
	getMetricDefinitionsResponse armmonitor.MetricDefinitionCollection
	getMetricDefinitionsErr      error

	getMetricNamespacesCalls    int
	getMetricNamespacesResponse armmonitor.MetricNamespaceCollection
	getMetricNamespacesErr      error

	getMetricValuesCalls     int
	getMetricValuesResponse  []armmonitor.Metric
	getMetricValuesTimegrain string
	getMetricValuesErr       error

	queryResourcesCalls    int
	queryResourcesResponse []azmetrics.MetricData
	queryResourcesErr      error
}

func (s *recordingService) GetResourceDefinitionById(id string) (armresources.GenericResource, error) {
	s.getResourceDefinitionByIDCalls++
	return s.getResourceDefinitionByIDResponse, s.getResourceDefinitionByIDErr
}

func (s *recordingService) GetResourceDefinitions(id []string, group []string, rType string, query string) ([]*armresources.GenericResourceExpanded, error) {
	s.getResourceDefinitionsCalls++
	return s.getResourceDefinitionsResponse, s.getResourceDefinitionsErr
}

func (s *recordingService) GetMetricDefinitionsWithRetry(resourceId string, namespace string) (armmonitor.MetricDefinitionCollection, error) {
	s.getMetricDefinitionsCalls++
	return s.getMetricDefinitionsResponse, s.getMetricDefinitionsErr
}

func (s *recordingService) GetMetricNamespaces(resourceId string) (armmonitor.MetricNamespaceCollection, error) {
	s.getMetricNamespacesCalls++
	return s.getMetricNamespacesResponse, s.getMetricNamespacesErr
}

func (s *recordingService) GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]armmonitor.Metric, string, error) {
	s.getMetricValuesCalls++
	return s.getMetricValuesResponse, s.getMetricValuesTimegrain, s.getMetricValuesErr
}

func (s *recordingService) QueryResources(resourceIDs []string, subscriptionID string, namespace string, timegrain string, startTime string, endTime string, metricNames []string, aggregations string, filter string, location string) ([]azmetrics.MetricData, error) {
	s.queryResourcesCalls++
	return s.queryResourcesResponse, s.queryResourcesErr
}
