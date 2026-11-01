// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type scopeKey struct{ region, namespace string }
type scopeMetrics struct {
	discoveryCalls, discoveryErrors, discovered, lastDiscovered int64
	collections, collectionErrors, empty, events                int64
	discoveryDuration, collectionDuration                       int64
	lastSuccess                                                 int64
}
type apiMetrics struct {
	calls, errors, throttles int64
	duration                 metrics.Sample
}

// cloudwatchMetrics keeps bounded namespace/region records, including namespaces
// that have never produced an event. It owns no cursor and cannot change collection.
type cloudwatchMetrics struct {
	failureSequence, partialResults       int64
	mu                                    sync.Mutex
	period, latency, granularity          time.Duration
	linkedAccounts                        bool
	windowStart, windowEnd, successfulEnd time.Time
	errors, partial, queries, datapoints  int64
	scopes                                map[scopeKey]*scopeMetrics
	scopeKeys                             []scopeKey
	api                                   map[string]*apiMetrics
}

func newCloudwatchMetrics(reg *monitoring.Registry, config *aws.MetricSet, configs []Config) *cloudwatchMetrics {
	if reg == nil {
		return nil
	}
	m := &cloudwatchMetrics{
		period: config.Period, latency: config.Latency, granularity: config.DataGranularity, linkedAccounts: config.IncludeLinkedAccounts,
		scopes: make(map[scopeKey]*scopeMetrics), api: make(map[string]*apiMetrics),
	}
	namespaces := map[string]bool{"*": true}
	for _, config := range configs {
		if config.Namespace != "" {
			namespaces[config.Namespace] = true
		}
	}
	for _, region := range config.RegionsList {
		for namespace := range namespaces {
			key := scopeKey{region, namespace}
			m.scopes[key] = &scopeMetrics{}
			m.scopeKeys = append(m.scopeKeys, key)
		}
	}
	sort.Slice(m.scopeKeys, func(i, j int) bool {
		a, b := m.scopeKeys[i], m.scopeKeys[j]
		if a.region != b.region {
			return a.region < b.region
		}
		return a.namespace < b.namespace
	})
	monitoring.NewFunc(reg, "cloudwatch", m.report, monitoring.Report)
	return m
}

func (m *cloudwatchMetrics) beginWindow(start, end time.Time) int64 {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.windowStart, m.windowEnd = start, end
	return m.failureSequence
}

func (m *cloudwatchMetrics) endWindow(previousErrors int64, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.errors++
		m.failureSequence++
	}
	if m.failureSequence == previousErrors {
		m.successfulEnd = m.windowEnd
	} else {
		m.partial++
	}
}

func (m *cloudwatchMetrics) observeScope(region, namespace string, discovery bool, start time.Time, count int, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := scopeKey{region, namespace}
	scope := m.scopes[key]
	if scope == nil {
		scope = m.scopes[scopeKey{region, "*"}]
	}
	if scope == nil {
		return
	}
	duration := time.Since(start).Nanoseconds()
	if discovery {
		scope.discoveryCalls++
		scope.discoveryDuration += duration
		scope.discovered += int64(count)
		scope.lastDiscovered = int64(count)
		if err != nil {
			scope.discoveryErrors++
		}
	} else {
		scope.collections++
		scope.collectionDuration += duration
		scope.events += int64(count)
		if err != nil {
			scope.collectionErrors++
		} else {
			scope.lastSuccess = time.Now().UnixMilli()
			if count == 0 {
				scope.empty++
			}
		}
	}
	if err != nil {
		m.errors++
		m.failureSequence++
	}
}

func (m *cloudwatchMetrics) addQueries(queries int) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queries += int64(queries)
}

// instrument counts one SDK invocation per page, including its retry time.
// It does not count individual HTTP retry attempts.
func (m *cloudwatchMetrics) instrument(stack *middleware.Stack) error {
	if m == nil {
		return nil
	}
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("CloudwatchInputMetrics",
		func(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
			started := time.Now()
			out, metadata, err := next.HandleInitialize(ctx, in)
			operation := awsmiddleware.GetOperationName(ctx)
			switch operation {
			case "ListMetrics", "GetMetricData", "GetResources", "GetRestApis", "GetApis", "DescribeInstances", "DescribeDBInstances", "ListQueues", "GetQueueAttributes":
			default:
				operation = "Other"
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			metric := m.api[operation]
			if metric == nil {
				metric = &apiMetrics{duration: metrics.NewUniformSample(256)}
				m.api[operation] = metric
			}
			metric.calls++
			metric.duration.Update(time.Since(started).Nanoseconds())
			if err != nil {
				metric.errors++
				m.failureSequence++
				var apiErr smithy.APIError
				if errors.As(err, &apiErr) {
					switch apiErr.ErrorCode() {
					case "Throttling", "ThrottlingException", "TooManyRequestsException", "RequestLimitExceeded", "ThrottledException":
						metric.throttles++
					}
				}
			}
			return out, metadata, err
		}), middleware.After)
}

func (m *cloudwatchMetrics) report(_ monitoring.Mode, v monitoring.Visitor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v.OnRegistryStart()
	defer v.OnRegistryFinished()
	monitoring.ReportInt(v, "period_ms", m.period.Milliseconds())
	monitoring.ReportInt(v, "latency_ms", m.latency.Milliseconds())
	monitoring.ReportInt(v, "data_granularity_ms", m.granularity.Milliseconds())
	monitoring.ReportBool(v, "include_linked_accounts", m.linkedAccounts)
	monitoring.ReportInt(v, "collection_errors_total", m.errors)
	monitoring.ReportInt(v, "incomplete_fetches_total", m.partial)
	monitoring.ReportInt(v, "metric_queries_total", m.queries)
	monitoring.ReportInt(v, "datapoints_received_total", m.datapoints)
	monitoring.ReportInt(v, "incomplete_results_total", m.partialResults)
	if !m.windowEnd.IsZero() {
		monitoring.ReportInt(v, "window_start", m.windowStart.UnixMilli())
		monitoring.ReportInt(v, "window_end", m.windowEnd.UnixMilli())
		_, target := aws.GetStartTimeEndTime(time.Now(), m.period, m.latency, time.Time{})
		monitoring.ReportInt(v, "collection_lag_ms", max(int64(0), target.Sub(m.windowEnd).Milliseconds()))
	}
	if !m.successfulEnd.IsZero() {
		monitoring.ReportInt(v, "last_successful_window_end", m.successfulEnd.UnixMilli())
		_, target := aws.GetStartTimeEndTime(time.Now(), m.period, m.latency, time.Time{})
		monitoring.ReportInt(v, "successful_collection_lag_ms", max(int64(0), target.Sub(m.successfulEnd).Milliseconds()))
	}
	monitoring.ReportNamespace(v, "scopes", func() {
		for i, key := range m.scopeKeys {
			s := m.scopes[key]
			monitoring.ReportNamespace(v, fmt.Sprintf("scope_%d", i), func() {
				monitoring.ReportString(v, "region", key.region)
				monitoring.ReportString(v, "namespace", key.namespace)
				monitoring.ReportInt(v, "discovery_calls_total", s.discoveryCalls)
				monitoring.ReportInt(v, "discovery_errors_total", s.discoveryErrors)
				monitoring.ReportInt(v, "metrics_discovered_total", s.discovered)
				monitoring.ReportInt(v, "metrics_discovered_last", s.lastDiscovered)
				monitoring.ReportInt(v, "discovery_duration_ns_total", s.discoveryDuration)
				monitoring.ReportInt(v, "collections_total", s.collections)
				monitoring.ReportInt(v, "collection_errors_total", s.collectionErrors)
				monitoring.ReportInt(v, "empty_collections_total", s.empty)
				monitoring.ReportInt(v, "events_created_total", s.events)
				monitoring.ReportInt(v, "collection_duration_ns_total", s.collectionDuration)
				monitoring.ReportInt(v, "last_success_at", s.lastSuccess)
			})
		}
	})
	monitoring.ReportNamespace(v, "api", func() {
		for name, a := range m.api {
			monitoring.ReportNamespace(v, name, func() {
				monitoring.ReportInt(v, "calls_total", a.calls)
				monitoring.ReportInt(v, "errors_total", a.errors)
				monitoring.ReportInt(v, "throttles_total", a.throttles)
				sample := a.duration.Snapshot()
				monitoring.ReportInt(v, "duration_ns_count", sample.Count())
				monitoring.ReportInt(v, "duration_ns_max", sample.Max())
				monitoring.ReportFloat(v, "duration_ns_mean", sample.Mean())
				monitoring.ReportFloat(v, "duration_ns_p95", sample.Percentile(0.95))
			})
		}
	})
}

func (m *cloudwatchMetrics) observeResults(results []cloudwatchtypes.MetricDataResult) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, result := range results {
		m.datapoints += int64(len(result.Values))
		switch result.StatusCode {
		case "", cloudwatchtypes.StatusCodeComplete:
		default:
			m.partialResults++
			m.failureSequence++
		}
	}
}
