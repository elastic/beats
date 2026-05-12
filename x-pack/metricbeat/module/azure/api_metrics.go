// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

type apiOperation string

const (
	operationGetResourceDefinitions    apiOperation = "get_resource_definitions"
	operationGetResourceDefinitionByID apiOperation = "get_resource_definition_by_id"
	operationGetMetricNamespaces       apiOperation = "get_metric_namespaces"
	operationGetMetricDefinitions      apiOperation = "get_metric_definitions"
	operationGetMetricValues           apiOperation = "get_metric_values"
	operationQueryResources            apiOperation = "query_resources"
)

type apiResult string

const (
	apiResultSuccess apiResult = "success"
	apiResultFailure apiResult = "failure"
)

type apiErrorKind string

const (
	apiErrorKindNone          apiErrorKind = "none"
	apiErrorKindThrottle      apiErrorKind = "throttle"
	apiErrorKindAzureResponse apiErrorKind = "azure_response"
	apiErrorKindTransport     apiErrorKind = "transport"
	apiErrorKindOther         apiErrorKind = "other"
)

type apiMetricKey struct {
	operation apiOperation
	result    apiResult
	errorKind apiErrorKind
}

type apiMetric struct {
	count    *monitoring.Uint
	duration metrics.Sample
}

type azureAPIMetrics struct {
	mu      sync.Mutex
	reg     *monitoring.Registry
	logger  *logp.Logger
	metrics map[apiMetricKey]*apiMetric
}

// newAzureAPIMetrics creates a new API metrics collector. It returns nil
// when reg is nil, making all observe calls a no-op.
func newAzureAPIMetrics(reg *monitoring.Registry, logger *logp.Logger) *azureAPIMetrics {
	if reg == nil {
		return nil
	}
	if logger == nil {
		logger = logp.NewNopLogger()
	}
	return &azureAPIMetrics{
		reg:     reg,
		logger:  logger,
		metrics: make(map[apiMetricKey]*apiMetric),
	}
}

func (m *azureAPIMetrics) observe(operation apiOperation, durationNanos int64, err error) {
	if m == nil {
		return
	}

	result := apiResultSuccess
	errorKind := apiErrorKindNone
	if err != nil {
		result = apiResultFailure
		errorKind = classifyAPIError(err)
	}

	key := apiMetricKey{
		operation: operation,
		result:    result,
		errorKind: errorKind,
	}

	m.mu.Lock()
	metric, ok := m.metrics[key]
	if !ok {
		metric = m.addMetric(operation, result, errorKind)
		m.metrics[key] = metric
	}
	m.mu.Unlock()

	metric.count.Add(1)
	metric.duration.Update(durationNanos)
}

// addMetric creates and registers a new metric for the given operation/result/errorKind.
// Must be called with m.mu held.
func (m *azureAPIMetrics) addMetric(operation apiOperation, result apiResult, errorKind apiErrorKind) *apiMetric {
	counterName := apiMetricName(operation, result, errorKind, "total")
	histogramName := apiMetricName(operation, result, errorKind, "duration")

	// Exponentially decaying reservoir so the histogram reflects recent API
	// behaviour rather than weighting hours-old observations equally with fresh
	// ones. Values are the Dropwizard ExponentiallyDecayingReservoir defaults:
	//   - 1028: reservoir size — enough samples for stable p50/p95/p99.
	//   - 0.015: decay rate — a sample weighs e^(-α·Δt), so one ~46s old is
	//     about half as important as one that just arrived, and anything older
	//     than ~10 min is effectively invisible.
	//
	// This fits Azure metricsets, which typically poll every 60s–5min: the
	// percentile histograms reflect the last few polling cycles, which is what
	// an operator asking "is Azure throttling us right now?" wants to see.
	duration := metrics.NewExpDecaySample(1028, 0.015)

	met := &apiMetric{
		count:    monitoring.NewUint(m.reg, counterName),
		duration: duration,
	}

	_ = adapter.GetGoMetrics(m.reg, histogramName, m.logger, adapter.Accept).
		GetOrRegister("histogram", metrics.NewHistogram(duration))

	return met
}

// apiMetricName returns a flat underscore-separated metric name for the
// given operation, result, error kind, and suffix.
//
// For success cases the error kind is omitted:
//
//	api_calls_get_metric_values_success_total
//	api_calls_get_metric_values_success_duration
//
// For failures the error kind is included:
//
//	api_calls_get_metric_values_failure_throttle_total
//	api_calls_get_metric_values_failure_throttle_duration
func apiMetricName(operation apiOperation, result apiResult, errorKind apiErrorKind, suffix string) string {
	name := fmt.Sprintf("api_calls_%s_%s", operation, result)
	if errorKind != apiErrorKindNone {
		name += "_" + string(errorKind)
	}
	return name + "_" + suffix
}

func classifyAPIError(err error) apiErrorKind {
	if err == nil {
		return apiErrorKindNone
	}

	var responseErr *azcore.ResponseError
	if errors.As(err, &responseErr) {
		if responseErr.StatusCode == 429 {
			return apiErrorKindThrottle
		}
		return apiErrorKindAzureResponse
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return apiErrorKindTransport
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return apiErrorKindTransport
	}

	return apiErrorKindOther
}
