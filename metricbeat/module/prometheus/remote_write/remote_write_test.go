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

package remote_write

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestGenerateEventsCounter tests counter simple cases
func TestGenerateEventsCounter(t *testing.T) {
	g := RemoteWriteEventGenerator{}

	timestamp := model.Time(424242)
	timestamp1 := model.Time(424243)
	labels := mapstr.M{
		"listener_name": model.LabelValue("http"),
	}

	// first fetch
	metrics := model.Samples{
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(42),
			Timestamp: timestamp,
		},
		&model.Sample{
			Metric: map[model.LabelName]model.LabelValue{
				"__name__":      "net_conntrack_listener_conn_closed_total",
				"listener_name": "http",
			},
			Value:     model.SampleValue(43),
			Timestamp: timestamp1,
		},
	}
	events := g.GenerateEvents(metrics)

	expected := mapstr.M{
		"metrics": mapstr.M{
			"net_conntrack_listener_conn_closed_total": float64(42),
		},
		"labels": labels,
	}
	expected1 := mapstr.M{
		"metrics": mapstr.M{
			"net_conntrack_listener_conn_closed_total": float64(43),
		},
		"labels": labels,
	}

	assert.Equal(t, len(events), 2)
	e := events[labels.String()+timestamp.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected)
	assert.EqualValues(t, e.Timestamp, timestamp.Time())
	e = events[labels.String()+timestamp1.Time().String()]
	assert.EqualValues(t, e.ModuleFields, expected1)
	assert.EqualValues(t, e.Timestamp, timestamp1.Time())
}

func TestMetricsCount(t *testing.T) {
	tests := []struct {
		name     string
		samples  model.Samples
		expected map[string]int
	}{
		{
			name: "HTTP requests counter with multiple dimensions",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "200", "path": "/api/v1/users"},
					Value:  100,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "POST", "status": "201", "path": "/api/v1/users"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "404", "path": "/api/v1/products"},
					Value:  10,
				},
			},
			expected: map[string]int{
				`{"method":"GET","path":"/api/v1/users","status":"200"}`:    1,
				`{"method":"POST","path":"/api/v1/users","status":"201"}`:   1,
				`{"method":"GET","path":"/api/v1/products","status":"404"}`: 1,
			},
		},
		{
			name: "CPU and memory usage gauges",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "node_cpu_usage_percent", "cpu": "0", "mode": "user"},
					Value:  25.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_cpu_usage_percent", "cpu": "0", "mode": "system"},
					Value:  10.2,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_memory_usage_bytes", "type": "used"},
					Value:  4294967296,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "node_memory_usage_bytes", "type": "free"},
					Value:  8589934592,
				},
			},
			expected: map[string]int{
				`{"cpu":"0","mode":"user"}`:   1,
				`{"cpu":"0","mode":"system"}`: 1,
				`{"type":"used"}`:             1,
				`{"type":"free"}`:             1,
			},
		},
		{
			name: "Request duration histogram",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "0.1", "handler": "/home"},
					Value:  200,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "0.5", "handler": "/home"},
					Value:  400,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_bucket", "le": "+Inf", "handler": "/home"},
					Value:  500,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_sum", "handler": "/home"},
					Value:  120.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "http_request_duration_seconds_count", "handler": "/home"},
					Value:  500,
				},
			},
			expected: map[string]int{
				`{"handler":"/home","le":"+Inf"}`: 1,
				`{"handler":"/home"}`:             2,
				`{"handler":"/home","le":"0.1"}`:  1,
				`{"handler":"/home","le":"0.5"}`:  1,
			},
		},
		{
			name: "Mix of counter, gauge, and histogram",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "http_requests_total", "method": "GET", "status": "200"},
					Value:  100,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "cpu_usage", "core": "0"},
					Value:  45.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_bucket", "le": "0.1"},
					Value:  30,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_bucket", "le": "0.5"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_sum"},
					Value:  75.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "request_duration_seconds_count"},
					Value:  60,
				},
			},
			expected: map[string]int{
				`{"le":"0.1"}`:                    1,
				`{"le":"0.5"}`:                    1,
				`{"method":"GET","status":"200"}`: 1,
				`{"core":"0"}`:                    1,
				`{}`:                              2,
			},
		},
		{
			name: "Duplicate labels and distinct labels",
			samples: model.Samples{
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/users", "method": "GET"},
					Value:  50,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/users", "method": "POST"},
					Value:  30,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "api_calls", "endpoint": "/products", "method": "GET"},
					Value:  40,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "system_load", "host": "server1"},
					Value:  1.5,
				},
				&model.Sample{
					Metric: model.Metric{"__name__": "system_load", "host": "server2"},
					Value:  2.0,
				},
			},
			expected: map[string]int{
				`{"endpoint":"/users","method":"GET"}`:    1,
				`{"endpoint":"/users","method":"POST"}`:   1,
				`{"endpoint":"/products","method":"GET"}`: 1,
				`{"host":"server1"}`:                      1,
				`{"host":"server2"}`:                      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := RemoteWriteEventGenerator{
				metricsCount: true,
			}

			events := generator.GenerateEvents(tt.samples)

			assert.Equal(t, len(tt.expected), len(events), "Number of generated events should match expected")

			for _, event := range events {
				count, ok := event.RootFields["metrics_count"]
				assert.True(t, ok, "metrics_count should be present for test: %s", tt.name)

				labels, ok := event.ModuleFields["labels"].(mapstr.M)
				if !ok {
					labels = mapstr.M{} // If no labels, create an empty map so that we can handle metrics with no labels
				}
				labelsHash := labels.String()

				expectedCount, ok := tt.expected[labelsHash]
				assert.True(t, ok, "should have an expected count for these labels")
				assert.Equal(t, expectedCount, count, "metrics_count should match expected value for labels %v", labels)

			}
		})
	}
}

// createTestWriteRequest creates a prompb.WriteRequest with the given number of samples
func createTestWriteRequest(numSamples int) *prompb.WriteRequest {
	samples := make([]prompb.Sample, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = prompb.Sample{
			Value:     float64(i),
			Timestamp: int64(i * 1000),
		}
	}

	return &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "test_metric"},
					{Name: "instance", Value: "localhost:9090"},
				},
				Samples: samples,
			},
		},
	}
}

// encodeWriteRequest encodes a WriteRequest to snappy-compressed protobuf
func encodeWriteRequest(req *prompb.WriteRequest) ([]byte, error) {
	data, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	return snappy.Encode(nil, data), nil
}

// newTestMetricSet creates a MetricSet for testing using the mbtest infrastructure
// to ensure proper initialization (including logger)
func newTestMetricSet(t *testing.T, maxCompressedBodyBytes, maxDecodedBodyBytes int64) *MetricSet {
	config := map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"remote_write"},
	}

	ms := mbtest.NewMetricSet(t, config)
	m, ok := ms.(*MetricSet)
	require.True(t, ok, "expected *MetricSet, got %T", ms)

	// Override the size limits for testing
	m.maxCompressedBodyBytes = maxCompressedBodyBytes
	m.maxDecodedBodyBytes = maxDecodedBodyBytes
	// Ensure events channel exists for the handler
	m.events = make(chan mb.Event, 100)

	return m
}

func TestHandleFuncDecodedSizeLimit(t *testing.T) {
	tests := []struct {
		name                   string
		maxDecodedBodyBytes    int64
		maxCompressedBodyBytes int64
		numSamples             int
		expectedStatus         int
		expectedBodyContains   string
	}{
		{
			name:                   "request within decoded size limit succeeds",
			maxDecodedBodyBytes:    1024 * 1024, // 1MB
			maxCompressedBodyBytes: 1024 * 1024, // 1MB
			numSamples:             10,
			expectedStatus:         http.StatusAccepted,
		},
		{
			name:                   "request exceeding decoded size limit rejected",
			maxDecodedBodyBytes:    100, // Very small limit
			maxCompressedBodyBytes: 1024 * 1024,
			numSamples:             100, // Will decode to more than 100 bytes
			expectedStatus:         http.StatusRequestEntityTooLarge,
			expectedBodyContains:   "decoded length too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMetricSet(t, tt.maxCompressedBodyBytes, tt.maxDecodedBodyBytes)

			// Create a test write request
			writeReq := createTestWriteRequest(tt.numSamples)
			body, err := encodeWriteRequest(writeReq)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			// Call the handler
			m.handleFunc(rec, req)

			// Check the response
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBodyContains != "" {
				assert.True(t, strings.Contains(rec.Body.String(), tt.expectedBodyContains),
					"expected body to contain %q, got %q", tt.expectedBodyContains, rec.Body.String())
			}
		})
	}
}

func TestHandleFuncCompressedSizeLimit(t *testing.T) {
	tests := []struct {
		name                   string
		maxCompressedBodyBytes int64
		maxDecodedBodyBytes    int64
		bodySize               int
		expectedStatus         int
		expectedBodyContains   string
	}{
		{
			name:                   "compressed body within limit succeeds",
			maxCompressedBodyBytes: 1024 * 1024, // 1MB
			maxDecodedBodyBytes:    10 * 1024 * 1024,
			bodySize:               100,
			expectedStatus:         http.StatusAccepted,
		},
		{
			name:                   "compressed body exceeding limit rejected",
			maxCompressedBodyBytes: 50,
			maxDecodedBodyBytes:    10 * 1024 * 1024,
			bodySize:               100, // More than 50 bytes
			expectedStatus:         http.StatusRequestEntityTooLarge,
			expectedBodyContains:   "request body too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMetricSet(t, tt.maxCompressedBodyBytes, tt.maxDecodedBodyBytes)

			var body []byte
			if tt.bodySize <= 100 {
				// For small sizes, use a valid request
				writeReq := createTestWriteRequest(tt.bodySize)
				var err error
				body, err = encodeWriteRequest(writeReq)
				require.NoError(t, err)
			} else {
				// For larger sizes, create arbitrary data
				body = make([]byte, tt.bodySize)
				for i := range body {
					body[i] = byte(i % 256)
				}
			}

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			// Call the handler
			m.handleFunc(rec, req)

			// Check the response
			assert.Equal(t, tt.expectedStatus, rec.Code)
			if tt.expectedBodyContains != "" {
				assert.True(t, strings.Contains(rec.Body.String(), tt.expectedBodyContains),
					"expected body to contain %q, got %q", tt.expectedBodyContains, rec.Body.String())
			}
		})
	}
}

func TestHandleFuncInvalidSnappyData(t *testing.T) {
	m := newTestMetricSet(t, 1024*1024, 10*1024*1024)

	// Send data with an invalid truncated varint header that will fail at snappy.DecodedLen. We simulate only one sample scenario.
	// A byte with high bit set (0x80+) indicates continuation, but with no following byte it's invalid
	invalidData := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(invalidData))
	rec := httptest.NewRecorder()

	m.handleFunc(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.True(t, strings.Contains(rec.Body.String(), "Decoded length error"),
		"expected 'Decoded length error' error, got %q", rec.Body.String())
}
