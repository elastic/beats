// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build !integration

package collector

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	pl "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus"
)

func TestGetPromEventsFromMetricFamily(t *testing.T) {
	labels := mapstr.M{
		"handler": "query",
	}
	tests := []struct {
		Family *p.MetricFamily
		Event  []PromEvent
	}{
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeCounter,
				Metric: []*p.OpenMetric{
					{
						Label: []*pl.Label{
							{
								Name:  "handler",
								Value: "query",
							},
						},
						Counter: &p.Counter{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: labels,
				},
			},
		},
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeGauge,
				Metric: []*p.OpenMetric{
					{
						Gauge: &p.Gauge{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: mapstr.M{},
				},
			},
		},
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeSummary,
				Metric: []*p.OpenMetric{
					{
						Summary: &p.Summary{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Quantile: []*p.Quantile{
								{
									Quantile: proto.Float64(0.99),
									Value:    proto.Float64(10),
								},
							},
						},
					},
				},
			},
			Event: []PromEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Labels: mapstr.M{},
				},
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: mapstr.M{
						"quantile": "0.99",
					},
				},
			},
		},
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeHistogram,
				Metric: []*p.OpenMetric{
					{
						Histogram: &p.Histogram{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Bucket: []*p.Bucket{
								{
									UpperBound:      proto.Float64(0.99),
									CumulativeCount: proto.Uint64(10),
								},
							},
						},
					},
				},
			},
			Event: []PromEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Labels: mapstr.M{},
				},
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_bucket": uint64(10),
						},
					},
					Labels: mapstr.M{"le": "0.99"},
				},
			},
		},
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeUnknown,
				Metric: []*p.OpenMetric{
					{
						Label: []*pl.Label{
							{
								Name:  "handler",
								Value: "query",
							},
						},
						Unknown: &p.Unknown{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: labels,
				},
			},
		},
	}

	p := promEventGenerator{}
	for _, test := range tests {
		event := p.GeneratePromEvents(test.Family)
		assert.Equal(t, test.Event, event)
	}
}

func TestSkipMetricFamily(t *testing.T) {
	testFamilies := []*p.MetricFamily{
		{
			Name: proto.String("http_request_duration_microseconds_a_a_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeCounter,
			Metric: []*p.OpenMetric{
				{
					Label: []*pl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Counter: &p.Counter{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_a_b_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeCounter,
			Metric: []*p.OpenMetric{
				{
					Label: []*pl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Counter: &p.Counter{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_b_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeGauge,
			Metric: []*p.OpenMetric{
				{
					Gauge: &p.Gauge{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_c_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeSummary,
			Metric: []*p.OpenMetric{
				{
					Summary: &p.Summary{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Quantile: []*p.Quantile{
							{
								Quantile: proto.Float64(0.99),
								Value:    proto.Float64(10),
							},
						},
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_d_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeHistogram,
			Metric: []*p.OpenMetric{
				{
					Histogram: &p.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*p.Bucket{
							{
								UpperBound:      proto.Float64(0.99),
								CumulativeCount: proto.Uint64(10),
							},
						},
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_e_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeUnknown,
			Metric: []*p.OpenMetric{
				{
					Label: []*pl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Unknown: &p.Unknown{
						Value: proto.Float64(10),
					},
				},
			},
		},
	}

	ms := &MetricSet{
		BaseMetricSet: mb.BaseMetricSet{},
	}

	// test with no filters
	ms.includeMetrics, _ = p.CompilePatternList(&[]string{})
	ms.excludeMetrics, _ = p.CompilePatternList(&[]string{})
	metricsToKeep := 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, metricsToKeep, len(testFamilies))

	// test with only one include filter
	ms.includeMetrics, _ = p.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	ms.excludeMetrics, _ = p.CompilePatternList(&[]string{})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, metricsToKeep, 2)

	// test with only one exclude filter
	ms.includeMetrics, _ = p.CompilePatternList(&[]string{""})
	ms.excludeMetrics, _ = p.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, len(testFamilies)-2, metricsToKeep)

	// test with ine include and one exclude
	ms.includeMetrics, _ = p.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	ms.excludeMetrics, _ = p.CompilePatternList(&[]string{"http_request_duration_microseconds_a_b_*"})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, 1, metricsToKeep)

}

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
	}

	expectedEvents := 11

	testCases := []struct {
		name                string
		expectedLabel       mapstr.M
		expectedMetricCount int64
	}{
		{"Prod API Inf", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "le": "+Inf", "service": "api"}, 1},
		{"Prod API 0.5", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "le": "0.5", "service": "api"}, 1},
		{"Prod API 1", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "le": "1", "service": "api"}, 1},
		{"Prod API Quantile 0.5", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.5", "service": "api"}, 1},
		{"Prod API Quantile 0.9", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.9", "service": "api"}, 1},
		{"Prod API Quantile 0.99", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "quantile": "0.99", "service": "api"}, 1},
		{"Prod API", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "service": "api"}, 6},
		{"Prod DB", mapstr.M{"environment": "prod", "instance": host, "job": "prometheus", "service": "db"}, 2},
		{"Staging API", mapstr.M{"environment": "staging", "instance": host, "job": "prometheus", "service": "api"}, 2},
		{"Staging DB", mapstr.M{"environment": "staging", "instance": host, "job": "prometheus", "service": "db"}, 2},
		{"Default", mapstr.M{"instance": host, "job": "prometheus"}, 1},
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
