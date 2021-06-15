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

package openmetricscollector

import (
	"testing"

	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"

	lbl "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"

	"github.com/elastic/beats/v7/metricbeat/helper/openmetrics"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	_ "github.com/elastic/beats/v7/metricbeat/module/prometheus"
)

func TestGetPromEventsFromMetricFamily(t *testing.T) {
	labels := common.MapStr{
		"handler": "query",
	}
	tests := []struct {
		Family *openmetrics.OpenMetricFamily
		Event  []OpenMetricEvent
	}{
		{
			Family: &openmetrics.OpenMetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeCounter,
				Metric: []*openmetrics.OpenMetric{
					{
						Name: proto.String("http_request_duration_microseconds_total"),
						Label: []*lbl.Label{
							{
								Name:  "handler",
								Value: "query",
							},
						},
						Counter: &openmetrics.Counter{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []OpenMetricEvent{
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_total": float64(10),
						},
					},
					Help:      "foo",
					Type:      textparse.MetricTypeCounter,
					Labels:    labels,
					Exemplars: common.MapStr{},
				},
			},
		},
		{
			Family: &openmetrics.OpenMetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeGauge,
				Metric: []*openmetrics.OpenMetric{
					{
						Gauge: &openmetrics.Gauge{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []OpenMetricEvent{
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeGauge,
					Labels: common.MapStr{},
				},
			},
		},
		{
			Family: &openmetrics.OpenMetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeSummary,
				Metric: []*openmetrics.OpenMetric{
					{
						Summary: &openmetrics.Summary{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Quantile: []*openmetrics.Quantile{
								{
									Quantile: proto.Float64(0.99),
									Value:    proto.Float64(10),
								},
							},
						},
					},
				},
			},
			Event: []OpenMetricEvent{
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeSummary,
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
			Family: &openmetrics.OpenMetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeHistogram,
				Metric: []*openmetrics.OpenMetric{
					{
						Histogram: &openmetrics.Histogram{
							SampleCount: proto.Uint64(10),
							SampleSum:   proto.Float64(10),
							Bucket: []*openmetrics.Bucket{
								{
									UpperBound:      proto.Float64(0.99),
									CumulativeCount: proto.Uint64(10),
								},
							},
						},
					},
				},
			},
			Event: []OpenMetricEvent{
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeHistogram,
					Labels: common.MapStr{},
				},
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds_bucket": uint64(10),
						},
					},
					Labels:    common.MapStr{"le": "0.99"},
					Exemplars: common.MapStr{},
				},
			},
		},
		{
			Family: &openmetrics.OpenMetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeUnknown,
				Metric: []*openmetrics.OpenMetric{
					{
						Label: []*lbl.Label{
							{
								Name:  "handler",
								Value: "query",
							},
						},
						Unknown: &openmetrics.Unknown{
							Value: proto.Float64(10),
						},
					},
				},
			},
			Event: []OpenMetricEvent{
				{
					Data: common.MapStr{
						"metrics": common.MapStr{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeUnknown,
					Labels: labels,
				},
			},
		},
	}

	p := openmetricEventGenerator{}
	for _, test := range tests {
		event := p.GenerateOpenMetricsEvents(test.Family)
		assert.Equal(t, test.Event, event)
	}
}

func TestSkipMetricFamily(t *testing.T) {
	testFamilies := []*openmetrics.OpenMetricFamily{
		{
			Name: proto.String("http_request_duration_microseconds_a_a_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeCounter,
			Metric: []*openmetrics.OpenMetric{
				{
					Label: []*lbl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Counter: &openmetrics.Counter{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_a_b_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeCounter,
			Metric: []*openmetrics.OpenMetric{
				{
					Label: []*lbl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Counter: &openmetrics.Counter{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_b_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeGauge,
			Metric: []*openmetrics.OpenMetric{
				{
					Gauge: &openmetrics.Gauge{
						Value: proto.Float64(10),
					},
				},
			},
		},
		{
			Name: proto.String("http_request_duration_microseconds_c_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeSummary,
			Metric: []*openmetrics.OpenMetric{
				{
					Summary: &openmetrics.Summary{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Quantile: []*openmetrics.Quantile{
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
			Metric: []*openmetrics.OpenMetric{
				{
					Histogram: &openmetrics.Histogram{
						SampleCount: proto.Uint64(10),
						SampleSum:   proto.Float64(10),
						Bucket: []*openmetrics.Bucket{
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
			Metric: []*openmetrics.OpenMetric{
				{
					Label: []*lbl.Label{
						{
							Name:  "handler",
							Value: "query",
						},
					},
					Unknown: &openmetrics.Unknown{
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
	mbtest.TestDataFiles(t, "prometheus", "openmetricscollector", "application/openmetrics-text")
}
