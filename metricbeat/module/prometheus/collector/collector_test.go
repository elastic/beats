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
// +build !integration

package collector

import (
	"testing"

	"github.com/elastic/beats/v8/metricbeat/mb"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	p "github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	_ "github.com/elastic/beats/v8/metricbeat/module/prometheus"
)

func TestGetPromEventsFromMetricFamily(t *testing.T) {
	labels := common.MapStr{
		"handler": "query",
	}
	tests := []struct {
		Family *dto.MetricFamily
		Event  []PromEvent
	}{
		{
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
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: labels,
				},
			},
		},
		{
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
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: common.MapStr{},
				},
			},
		},
		{
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
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Labels: common.MapStr{},
				},
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Labels: common.MapStr{
						"quantile": "0.99",
					},
				},
			},
		},
		{
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
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Labels: common.MapStr{},
				},
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_bucket": uint64(10),
						},
					},
					Labels: common.MapStr{"le": "0.99"},
				},
			},
		},
		{
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
					Data: common.MapStr{
						"metrics": common.MapStr{
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
	testFamilies := []*dto.MetricFamily{
		{
			Name: proto.String("http_request_duration_microseconds_a_a_in"),
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
		{
			Name: proto.String("http_request_duration_microseconds_a_b_in"),
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
		{
			Name: proto.String("http_request_duration_microseconds_b_in"),
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
		{
			Name: proto.String("http_request_duration_microseconds_c_in"),
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
		{
			Name: proto.String("http_request_duration_microseconds_d_in"),
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
		{
			Name: proto.String("http_request_duration_microseconds_e_in"),
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
