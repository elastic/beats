// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package collector

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "prometheus", "collector")
}

func sortPromEvents(events []mb.Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].RootFields["prometheus"].(mapstr.M)["labels"].(mapstr.M).String() < events[j].RootFields["prometheus"].(mapstr.M)["labels"].(mapstr.M).String()
	})
}

// TestFetchEventForCountingMetrics tests the functionality of fetching events for counting metrics in the Prometheus collector.
// NOTE: For the remote_write metricset, the test will be similar. So, we will only test this for the collector metricset.
func TestFetchEventForCountingMetrics(t *testing.T) {
	metricsPath := "/metrics"
	server := initServer(metricsPath)
	defer server.Close()

	host := strings.TrimPrefix(server.URL, "http://")

	config := map[string]interface{}{
		"module":        "prometheus",
		"metricsets":    []string{"collector"},
		"hosts":         []string{server.URL},
		"metrics_path":  metricsPath,
		"metrics_count": true,
		"use_types":     true,
		"rate_counters": true,
	}

	expectedEvents := 8

	testCases := []struct {
		name                string
		expectedLabel       mapstr.M
		expectedMetricCount int64
	}{
		{"ProdAPIWithQuantile50", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.5", "service": "api"}, 1},
		{"ProdAPIWithQuantile90", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.9", "service": "api"}, 1},
		{"ProdAPIWithQuantile99", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.99", "service": "api"}, 1},
		{"ProdAPIWithoutQuantile", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "service": "api"}, 5},
		{"ProdDBWithoutQuantile", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "service": "db"}, 2},
		{"StagingAPIWithoutQuantile", mapstr.M{"environment": "staging", "instance": host, "job": "prometheus", "service": "api"}, 2},
		{"StagingDBWithoutQuantile", mapstr.M{"environment": "staging", "instance": host, "job": "prometheus", "service": "db"}, 2},
		{"PrometheusJobOnly", mapstr.M{"instance": host, "job": "prometheus"}, 1},
	}
	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)

	for _, err := range errs {
		t.Errorf("Unexpected error: %v", err)
	}

	assert.Equal(t, expectedEvents, len(events), "Number of events does not match expected")

	sortPromEvents(events)

	for i := range expectedEvents {
		t.Run(testCases[i].name, func(t *testing.T) {
			validateEvent(t, events[i], testCases[i].expectedLabel, testCases[i].expectedMetricCount)
		})
	}
}
func validateEvent(t *testing.T, event mb.Event, expectedLabels mapstr.M, expectedMetricsCount int64) {
	t.Helper()

	metricsCount, err := event.RootFields.GetValue("metrics_count")
	assert.NoError(t, err, "Failed to get metrics_count")

	labels, ok := event.RootFields["prometheus"].(mapstr.M)["labels"].(mapstr.M)
	assert.True(t, ok, "Failed to get labels")

	assert.Equal(t, expectedLabels, labels, "Labels do not match expected")
	assert.Equal(t, expectedMetricsCount, metricsCount, "Metrics count does not match expected")
}

func initServer(endpoint string) *httptest.Server {
	data := []byte(`# HELP test_gauge A test gauge metric
# TYPE test_gauge gauge
test_gauge{environment="prod",service="api"} 10.5
test_gauge{environment="staging",service="api"} 8.2
test_gauge{environment="prod",service="db"} 20.7
test_gauge{environment="staging",service="db"} 15.1

# HELP test_counter A test counter metric
# TYPE test_counter counter
test_counter{environment="prod",service="api"} 42
test_counter{environment="staging",service="api"} 444
test_counter{environment="prod",service="db"} 123
test_counter{environment="staging",service="db"} 98

# HELP test_histogram A test histogram metric
# TYPE test_histogram histogram
test_histogram_bucket{environment="prod",service="api",le="0.1"} 0
test_histogram_bucket{environment="prod",service="api",le="0.5"} 1
test_histogram_bucket{environment="prod",service="api",le="1.0"} 2
test_histogram_bucket{environment="prod",service="api",le="+Inf"} 3
test_histogram_sum{environment="prod",service="api"} 2.7
test_histogram_count{environment="prod",service="api"} 3

# HELP test_summary A test summary metric
# TYPE test_summary summary
test_summary{environment="prod",service="api",quantile="0.5"} 0.2
test_summary{environment="prod",service="api",quantile="0.9"} 0.7
test_summary{environment="prod",service="api",quantile="0.99"} 1.2
test_summary_sum{environment="prod",service="api"} 1234.5
test_summary_count{environment="prod",service="api"} 1000`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == endpoint {
			// https://github.com/prometheus/client_golang/blob/dbf72fc1a20e87bea6e15281eda7ef4d139a01ec/prometheus/registry_test.go#L364
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server
}
