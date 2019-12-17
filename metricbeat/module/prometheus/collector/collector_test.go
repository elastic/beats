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

// +build !integration

package collector

import (
	"testing"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	_ "github.com/elastic/beats/metricbeat/module/prometheus"
)

func TestGetPromEventsFromMetricFamily(t *testing.T) {
	labels := common.MapStr{
		"handler": "query",
	}
	tests := []struct {
		Name         string
		Family       *dto.MetricFamily
		Event        []PromEvent
		CounterCache prometheus.CounterCache
	}{
		{
			Name: "Parse counter",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_COUNTER.Enum(),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  proto.String("handler"),
								Value: proto.String("query"),
							},
						},
						Counter: &dto.Counter{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(10),
					},
					labels: labels,
				},
			},
		},
		{
			Name: "Parse gauge",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_GAUGE.Enum(),
				Metric: []*dto.Metric{
					{
						Gauge: &dto.Gauge{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(10),
					},
					labels: common.MapStr{},
				},
			},
		},
		{
			Name: "Parse summary",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_SUMMARY.Enum(),
				Metric: []*dto.Metric{
					{
						Summary: &dto.Summary{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Quantile: []*dto.Quantile{
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
					data: common.MapStr{
						"http_request_duration_microseconds_count": uint64(10),
						"http_request_duration_microseconds_sum":   float64(10),
					},
					labels: common.MapStr{},
				},
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(10),
					},
					labels: common.MapStr{
						"quantile": "0.99",
					},
				},
			},
		},
		{
			Name: "Parse histogram",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_HISTOGRAM.Enum(),
				Metric: []*dto.Metric{
					{
						Histogram: &dto.Histogram{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Bucket: []*dto.Bucket{
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
					data: common.MapStr{
						"http_request_duration_microseconds_count": uint64(10),
						"http_request_duration_microseconds_sum":   float64(10),
					},
					labels: common.MapStr{},
				},
				{
					data: common.MapStr{
						"http_request_duration_microseconds_bucket": uint64(10),
					},
					labels: common.MapStr{"le": "0.99"},
				},
			},
		},
		{
			Name: "Parse untyped",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_UNTYPED.Enum(),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  proto.String("handler"),
								Value: proto.String("query"),
							},
						},
						Untyped: &dto.Untyped{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(10),
					},
					labels: labels,
				},
			},
		},
		{
			Name: "Parse counter with rate enabled",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_COUNTER.Enum(),
				Metric: []*dto.Metric{
					{
						Label: []*dto.LabelPair{
							{
								Name:  proto.String("handler"),
								Value: proto.String("query"),
							},
						},
						Counter: &dto.Counter{
							Value: proto.Float64(10),
						},
					},
				},
			},
			CounterCache: &mockCounter{
				counters: map[string]float64{
					"http_request_duration_microseconds{\"handler\":\"query\"}": 3,
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(7),
					},
					labels: labels,
				},
			},
		},
		{
			Name: "Parse summary with rate enabled",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_SUMMARY.Enum(),
				Metric: []*dto.Metric{
					{
						Summary: &dto.Summary{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Quantile: []*dto.Quantile{
								{
									Quantile: proto.Float64(0.99),
									Value:    proto.Float64(10),
								},
							},
						},
					},
				},
			},
			CounterCache: &mockCounter{
				counters: map[string]float64{
					"http_request_duration_microseconds_count{}":                7,
					"http_request_duration_microseconds_sum{}":                  5,
					"http_request_duration_microseconds{\"quantile\":\"0.99\"}": 3,
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds_count": uint64(3),
						"http_request_duration_microseconds_sum":   float64(5),
					},
					labels: common.MapStr{},
				},
				{
					data: common.MapStr{
						"http_request_duration_microseconds": float64(7),
					},
					labels: common.MapStr{
						"quantile": "0.99",
					},
				},
			},
		},
		{
			Name: "Parse histogram with rate enabled",
			Family: &dto.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: dto.MetricType_HISTOGRAM.Enum(),
				Metric: []*dto.Metric{
					{
						Histogram: &dto.Histogram{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Bucket: []*dto.Bucket{
								{
									UpperBound:      proto.Float64(0.99),
									CumulativeCount: proto.Uint64(10),
								},
							},
						},
					},
				},
			},
			CounterCache: &mockCounter{
				counters: map[string]float64{
					"http_request_duration_microseconds_count{}":                 8,
					"http_request_duration_microseconds_sum{}":                   7,
					"http_request_duration_microseconds_bucket{\"le\":\"0.99\"}": 1,
				},
			},
			Event: []PromEvent{
				{
					data: common.MapStr{
						"http_request_duration_microseconds_count": uint64(2),
						"http_request_duration_microseconds_sum":   float64(3),
					},
					labels: common.MapStr{},
				},
				{
					data: common.MapStr{
						"http_request_duration_microseconds_bucket": uint64(9),
					},
					labels: common.MapStr{"le": "0.99"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			event := getPromEventsFromMetricFamily(test.Family, test.CounterCache)
			assert.Equal(t, test.Event, event)
		})
	}
}

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "prometheus", "collector")
}

type mockCounter struct {
	counters map[string]float64
}

func (c *mockCounter) Start() {}
func (c *mockCounter) Stop()  {}

func (c *mockCounter) RateUint64(counterName string, value uint64) uint64 {
	return value - uint64(c.counters[counterName])
}

func (c *mockCounter) RateFloat64(counterName string, value float64) float64 {
	return value - c.counters[counterName]
}
