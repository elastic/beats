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
