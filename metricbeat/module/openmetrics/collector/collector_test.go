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

	prometheuslabels "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/textparse"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	"github.com/elastic/beats/v7/metricbeat/helper/openmetrics"
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	_ "github.com/elastic/beats/v7/metricbeat/module/openmetrics"
)

func TestData(t *testing.T) {
	mbtest.TestDataFiles(t, "openmetrics", "collector")
}

func TestSameLabels(t *testing.T) {
	dataConfig := mbtest.ReadDataConfig(t, "_meta/samelabeltestdata/config.yml")
	mbtest.TestDataFilesWithConfig(t, "openmetrics", "collector", dataConfig, "")
}

func TestGetOpenMetricsEventsFromMetricFamily(t *testing.T) {
	labels := mapstr.M{
		"handler": "query",
	}
	tests := []struct {
		Family *p.MetricFamily
		Event  []OpenMetricEvent
	}{
		{
			Family: &p.MetricFamily{
				Name: proto.String("http_request_duration_microseconds"),
				Help: proto.String("foo"),
				Type: textparse.MetricTypeCounter,
				Metric: []*p.OpenMetric{
					{
						Name: proto.String("http_request_duration_microseconds_total"),
						Label: []*prometheuslabels.Label{
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
			Event: []OpenMetricEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_total": float64(10),
						},
					},
					Help:      "foo",
					Type:      textparse.MetricTypeCounter,
					Labels:    labels,
					Exemplars: mapstr.M{},
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
			Event: []OpenMetricEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds": float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeGauge,
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
			Event: []OpenMetricEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeSummary,
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
			Event: []OpenMetricEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_count": uint64(10),
							"http_request_duration_microseconds_sum":   float64(10),
						},
					},
					Help:   "foo",
					Type:   textparse.MetricTypeHistogram,
					Labels: mapstr.M{},
				},
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
							"http_request_duration_microseconds_bucket": uint64(10),
						},
					},
					Labels:    mapstr.M{"le": "0.99"},
					Exemplars: mapstr.M{},
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
						Label: []*prometheuslabels.Label{
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
			Event: []OpenMetricEvent{
				{
					Data: mapstr.M{
						"metrics": mapstr.M{
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
	testFamilies := []*p.MetricFamily{
		{
			Name: proto.String("http_request_duration_microseconds_a_a_in"),
			Help: proto.String("foo"),
			Type: textparse.MetricTypeCounter,
			Metric: []*p.OpenMetric{
				{
					Label: []*prometheuslabels.Label{
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
					Label: []*prometheuslabels.Label{
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
					Label: []*prometheuslabels.Label{
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
	ms.includeMetrics, _ = openmetrics.CompilePatternList(&[]string{})
	ms.excludeMetrics, _ = openmetrics.CompilePatternList(&[]string{})
	metricsToKeep := 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, metricsToKeep, len(testFamilies))

	// test with only one include filter
	ms.includeMetrics, _ = openmetrics.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	ms.excludeMetrics, _ = openmetrics.CompilePatternList(&[]string{})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, metricsToKeep, 2)

	// test with only one exclude filter
	ms.includeMetrics, _ = openmetrics.CompilePatternList(&[]string{""})
	ms.excludeMetrics, _ = openmetrics.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, len(testFamilies)-2, metricsToKeep)

	// test with one include and one exclude
	ms.includeMetrics, _ = openmetrics.CompilePatternList(&[]string{"http_request_duration_microseconds_a_*"})
	ms.excludeMetrics, _ = openmetrics.CompilePatternList(&[]string{"http_request_duration_microseconds_a_b_*"})
	metricsToKeep = 0
	for _, testFamily := range testFamilies {
		if !ms.skipFamily(testFamily) {
			metricsToKeep++
		}
	}
	assert.Equal(t, 1, metricsToKeep)

}
