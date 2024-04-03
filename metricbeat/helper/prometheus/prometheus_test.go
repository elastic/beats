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

package prometheus

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	promMetrics = `
# TYPE first_metric gauge
first_metric{label1="value1",label2="value2",label3="Value3",label4="FOO"} 1
# TYPE second_metric gauge
second_metric{label1="value1",label3="othervalue"} 0
# TYPE summary_metric summary
summary_metric{quantile="0.5"} 29735
summary_metric{quantile="0.9"} 47103
summary_metric{quantile="0.99"} 50681
summary_metric_sum 234892394
summary_metric_count 44000
# TYPE histogram_metric histogram
histogram_metric_bucket{le="1000"} 1
histogram_metric_bucket{le="10000"} 1
histogram_metric_bucket{le="100000"} 1
histogram_metric_bucket{le="1e+06"} 1
histogram_metric_bucket{le="1e+08"} 1
histogram_metric_bucket{le="1e+09"} 1
histogram_metric_bucket{le="+Inf"} 1
histogram_metric_sum 117
histogram_metric_count 1
# TYPE histogram_decimal_metric histogram
histogram_decimal_metric_bucket{le="0.001"} 1
histogram_decimal_metric_bucket{le="0.01"} 1
histogram_decimal_metric_bucket{le="0.1"} 2
histogram_decimal_metric_bucket{le="1"} 3
histogram_decimal_metric_bucket{le="+Inf"} 5
histogram_decimal_metric_sum 4.31
histogram_decimal_metric_count 5

`

	promInfoMetrics = `
# TYPE target info
target_info 1
# TYPE first_metric gauge
first_metric{label1="value1",label2="value2",label3="Value3",label4="FOO"} 1

`

	promGaugeKeyLabel = `
# TYPE metrics_one_count_total gauge
metrics_one_count_total{name="jane",surname="foster"} 1
metrics_one_count_total{name="john",surname="williams"} 2
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3

`

	promGaugeKeyLabelWithNaNInf = `
# TYPE metrics_one_count_errors gauge
metrics_one_count_errors{name="jane",surname="foster"} 0
# TYPE metrics_one_count_total gauge
metrics_one_count_total{name="jane",surname="foster"} NaN
metrics_one_count_total{name="foo",surname="bar"} +Inf
metrics_one_count_total{name="john",surname="williams"} -Inf
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3

`

	promCounterKeyLabel = `
# TYPE metrics_one_count_total counter
metrics_one_count_total{name="jane",surname="foster"} 1
metrics_one_count_total{name="john",surname="williams"} 2
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3

`

	promCounterKeyLabelWithNaNInf = `
# TYPE metrics_one_count_errors counter
metrics_one_count_errors{name="jane",surname="foster"} 1
# TYPE metrics_one_count_total counter
metrics_one_count_total{name="jane",surname="foster"} NaN
metrics_one_count_total{name="john",surname="williams"} +Inf
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3

`

	promHistogramKeyLabel = `
# TYPE metrics_one_midichlorians histogram
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="2000"} 52
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="4000"} 70
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="8000"} 78
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="16000"} 84
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="32000"} 86
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="+Inf"} 86
metrics_one_midichlorians_sum{rank="youngling",alive="yes"} 1000001
metrics_one_midichlorians_count{rank="youngling",alive="yes"} 86
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="2000"} 16
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="4000"} 20
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="8000"} 23
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="16000"} 27
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="32000"} 27
metrics_one_midichlorians_bucket{rank="padawan",alive="yes",le="+Inf"} 28
metrics_one_midichlorians_sum{rank="padawan",alive="yes"} 800001
metrics_one_midichlorians_count{rank="padawan",alive="yes"} 28

`

	promHistogramKeyLabelWithNaNInf = `
# TYPE metrics_one_midichlorians histogram
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="2000"} NaN
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="4000"} +Inf
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="8000"} -Inf
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="16000"} 84
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="32000"} 86
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="+Inf"} 86
metrics_one_midichlorians_sum{rank="youngling",alive="yes"} 1000001
metrics_one_midichlorians_count{rank="youngling",alive="yes"} 86

`

	promSummaryKeyLabel = `
# TYPE metrics_force_propagation_ms summary
metrics_force_propagation_ms{kind="jedi",quantile="0"} 35
metrics_force_propagation_ms{kind="jedi",quantile="0.25"} 22
metrics_force_propagation_ms{kind="jedi",quantile="0.5"} 7
metrics_force_propagation_ms{kind="jedi",quantile="0.75"} 20
metrics_force_propagation_ms{kind="jedi",quantile="1"} 30
metrics_force_propagation_ms_sum{kind="jedi"} 89
metrics_force_propagation_ms_count{kind="jedi"} 651
metrics_force_propagation_ms{kind="sith",quantile="0"} 30
metrics_force_propagation_ms{kind="sith",quantile="0.25"} 20
metrics_force_propagation_ms{kind="sith",quantile="0.5"} 12
metrics_force_propagation_ms{kind="sith",quantile="0.75"} 21
metrics_force_propagation_ms{kind="sith",quantile="1"} 29
metrics_force_propagation_ms_sum{kind="sith"} 112
metrics_force_propagation_ms_count{kind="sith"} 711

`

	promSummaryKeyLabelWithNaNInf = `
# TYPE metrics_force_propagation_ms summary
metrics_force_propagation_ms{kind="jedi",quantile="0"} NaN
metrics_force_propagation_ms{kind="jedi",quantile="0.25"} +Inf
metrics_force_propagation_ms{kind="jedi",quantile="0.5"} -Inf
metrics_force_propagation_ms{kind="jedi",quantile="0.75"} 20
metrics_force_propagation_ms{kind="jedi",quantile="1"} 30
metrics_force_propagation_ms_sum{kind="jedi"} 50
metrics_force_propagation_ms_count{kind="jedi"} 651

`

	promGaugeLabeled = `
# TYPE metrics_that_inform_labels gauge
metrics_that_inform_labels{label1="I am 1", label2="I am 2"} 1
metrics_that_inform_labels{label1="I am 1", label3="I am 3"} 1
# TYPE metrics_that_use_labels gauge
metrics_that_use_labels{label1="I am 1"} 20

`
)

type mockFetcher struct {
	response string
}

var _ = httpfetcher(&mockFetcher{})

// FetchResponse returns an HTTP response but for the Body, which
// returns the mockFetcher.Response contents
func (m mockFetcher) FetchResponse() (*http.Response, error) {
	body := bytes.NewBuffer(nil)
	writer := gzip.NewWriter(body)
	_, _ = writer.Write([]byte(m.response))
	writer.Close()

	return &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
			"Content-Type":     []string{"text/plain; version=0.0.4; charset=utf-8"},
		},
		Body: io.NopCloser(body),
	}, nil
}

func TestPrometheus(t *testing.T) {

	p := &prometheus{mockFetcher{response: promMetrics}, logp.NewLogger("test")}

	tests := []struct {
		mapping  *MetricsMapping
		msg      string
		expected []mapstr.M
	}{
		{
			msg: "Simple field map",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": Metric("first.metric"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
				},
			},
		},
		{
			msg: "Simple field map with labels",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": Metric("first.metric"),
				},
				Labels: map[string]LabelMap{
					"label1": Label("labels.label1"),
					"label2": Label("labels.label2"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
					"labels": mapstr.M{
						"label1": "value1",
						"label2": "value2",
					},
				},
			},
		},
		{
			msg: "Several metrics",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric":  Metric("first.metric"),
					"second_metric": Metric("second.metric"),
				},
				Labels: map[string]LabelMap{
					"label3": KeyLabel("labels.label3"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
					"labels": mapstr.M{
						"label3": "Value3",
					},
				},
				mapstr.M{
					"second": mapstr.M{
						"metric": 0.0,
					},
					"labels": mapstr.M{
						"label3": "othervalue",
					},
				},
			},
		},
		{
			msg: "Grouping by key labels",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric":  Metric("first.metric"),
					"second_metric": Metric("second.metric"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("labels.label1"),
					"label2": Label("labels.label2"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
					"second": mapstr.M{
						"metric": 0.0,
					},
					"labels": mapstr.M{
						"label1": "value1",
						"label2": "value2",
					},
				},
			},
		},
		{
			msg: "Keyword metrics",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric":  KeywordMetric("first.metric", "works"),
					"second_metric": KeywordMetric("second.metric", "itsnot"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("labels.label1"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": "works",
					},
					"labels": mapstr.M{
						"label1": "value1",
					},
				},
			},
		},
		{
			msg: "Boolean metrics",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric":  BooleanMetric("first.metric"),
					"second_metric": BooleanMetric("second.metric"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("labels.label1"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": true,
					},
					"second": mapstr.M{
						"metric": false,
					},
					"labels": mapstr.M{
						"label1": "value1",
					},
				},
			},
		},
		{
			msg: "Label metrics",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": LabelMetric("first.metric", "label3"),
				},
				Labels: map[string]LabelMap{
					"label1": Label("labels.label1"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": "Value3",
					},
					"labels": mapstr.M{
						"label1": "value1",
					},
				},
			},
		},
		{
			msg: "Label metrics, lowercase",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": LabelMetric("first.metric", "label4", OpLowercaseValue()),
				},
				Labels: map[string]LabelMap{
					"label1": Label("labels.label1"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": "foo",
					},
					"labels": mapstr.M{
						"label1": "value1",
					},
				},
			},
		},
		{
			msg: "Label metrics, filter",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": LabelMetric("first.metric", "label4", OpFilterMap(
						"label1",
						map[string]string{"value1": "foo"},
					)),
				},
				Labels: map[string]LabelMap{
					"label1": Label("labels.label1"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": mapstr.M{
							"foo": "FOO",
						},
					},
					"labels": mapstr.M{
						"label1": "value1",
					},
				},
			},
		},
		{
			msg: "Label metrics, filter",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": LabelMetric("first.metric", "label4", OpLowercaseValue(), OpFilterMap(
						"foo",
						map[string]string{"Filtered": "filtered"},
					)),
				},
				Labels: map[string]LabelMap{
					"label1": Label("labels.label1"),
				},
			},
			expected: []mapstr.M{},
		},
		{
			msg: "Summary metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"summary_metric": Metric("summary.metric"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"summary": mapstr.M{
						"metric": mapstr.M{
							"sum":   234892394.0,
							"count": uint64(44000),
							"percentile": mapstr.M{
								"50": 29735.0,
								"90": 47103.0,
								"99": 50681.0,
							},
						},
					},
				},
			},
		},
		{
			msg: "Histogram metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"histogram_metric": Metric("histogram.metric"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"histogram": mapstr.M{
						"metric": mapstr.M{
							"count": uint64(1),
							"bucket": mapstr.M{
								"1000000000": uint64(1),
								"+Inf":       uint64(1),
								"1000":       uint64(1),
								"10000":      uint64(1),
								"100000":     uint64(1),
								"1000000":    uint64(1),
								"100000000":  uint64(1),
							},
							"sum": 117.0,
						},
					},
				},
			},
		},
		{
			msg: "Histogram decimal metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"histogram_decimal_metric": Metric("histogram.metric", OpMultiplyBuckets(1000)),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"histogram": mapstr.M{
						"metric": mapstr.M{
							"count": uint64(5),
							"bucket": mapstr.M{
								"1":    uint64(1),
								"10":   uint64(1),
								"100":  uint64(2),
								"1000": uint64(3),
								"+Inf": uint64(5),
							},
							"sum": 4310.0,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			reporter := &mbtest.CapturingReporterV2{}
			_ = p.ReportProcessedMetrics(test.mapping, reporter)
			assert.Nil(t, reporter.GetErrors(), test.msg)
			// Sort slice to avoid randomness
			res := reporter.GetEvents()
			sort.Slice(res, func(i, j int) bool {
				return res[i].MetricSetFields.String() < res[j].MetricSetFields.String()
			})
			assert.Equal(t, len(test.expected), len(res))
			for j, ev := range res {
				assert.Equal(t, test.expected[j], ev.MetricSetFields, test.msg)
			}
		})
	}
}

// NOTE: if the content type = text/plain prometheus doesn't support Info metrics
// with the current implementation, info metrics should just be ignored and all other metrics
// correctly processed
func TestInfoMetricPrometheus(t *testing.T) {

	p := &prometheus{mockFetcher{response: promInfoMetrics}, logp.NewLogger("test")}

	tests := []struct {
		mapping  *MetricsMapping
		msg      string
		expected []mapstr.M
	}{
		{
			msg: "Ignore metrics not in mapping",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": Metric("first.metric"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
				},
			},
		},
		{
			msg: "Ignore metric in mapping but of unsupported type (eg. Info metric)",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": Metric("first.metric"),
					"target_info":  Metric("target.info"),
				},
			},
			expected: []mapstr.M{
				mapstr.M{
					"first": mapstr.M{
						"metric": 1.0,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			reporter := &mbtest.CapturingReporterV2{}
			_ = p.ReportProcessedMetrics(test.mapping, reporter)
			assert.Nil(t, reporter.GetErrors(), test.msg)
			// Sort slice to avoid randomness
			res := reporter.GetEvents()
			sort.Slice(res, func(i, j int) bool {
				return res[i].MetricSetFields.String() < res[j].MetricSetFields.String()
			})
			assert.Equal(t, len(test.expected), len(res))
			for j, ev := range res {
				assert.Equal(t, test.expected[j], ev.MetricSetFields, test.msg)
			}
		})
	}
}

func TestPrometheusKeyLabels(t *testing.T) {

	testCases := []struct {
		testName           string
		prometheusResponse string
		mapping            *MetricsMapping
		expectedEvents     []mapstr.M
	}{
		{
			testName:           "Test gauge with KeyLabel",
			prometheusResponse: promGaugeKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_count_total": Metric("metrics.one.count"),
				},
				Labels: map[string]LabelMap{
					"name":    KeyLabel("metrics.one.labels.name"),
					"surname": KeyLabel("metrics.one.labels.surname"),
					"age":     KeyLabel("metrics.one.labels.age"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": 1.0,
							"labels": mapstr.M{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": 2.0,
							"labels": mapstr.M{
								"name":    "john",
								"surname": "williams",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": 3.0,
							"labels": mapstr.M{
								"name":    "jahn",
								"surname": "baldwin",
								"age":     "30",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test gauge with KeyLabel With NaN Inf",
			prometheusResponse: promGaugeKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_count_errors": Metric("metrics.one.count"),
					"metrics_one_count_total":  Metric("metrics.one.count"),
				},
				Labels: map[string]LabelMap{
					"name":    KeyLabel("metrics.one.labels.name"),
					"surname": KeyLabel("metrics.one.labels.surname"),
					"age":     KeyLabel("metrics.one.labels.age"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": 0.0,
							"labels": mapstr.M{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": 3.0,
							"labels": mapstr.M{
								"name":    "jahn",
								"surname": "baldwin",
								"age":     "30",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test counter with KeyLabel",
			prometheusResponse: promCounterKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_count_total": Metric("metrics.one.count"),
				},
				Labels: map[string]LabelMap{
					"name":    KeyLabel("metrics.one.labels.name"),
					"surname": KeyLabel("metrics.one.labels.surname"),
					"age":     KeyLabel("metrics.one.labels.age"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": int64(1),
							"labels": mapstr.M{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": int64(2),
							"labels": mapstr.M{
								"name":    "john",
								"surname": "williams",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": int64(3),
							"labels": mapstr.M{
								"name":    "jahn",
								"surname": "baldwin",
								"age":     "30",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test counter with KeyLabel With NaN Inf",
			prometheusResponse: promCounterKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_count_errors": Metric("metrics.one.count"),
					"metrics_one_count_total":  Metric("metrics.one.count"),
				},
				Labels: map[string]LabelMap{
					"name":    KeyLabel("metrics.one.labels.name"),
					"surname": KeyLabel("metrics.one.labels.surname"),
					"age":     KeyLabel("metrics.one.labels.age"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": int64(1),
							"labels": mapstr.M{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"count": int64(3),
							"labels": mapstr.M{
								"name":    "jahn",
								"surname": "baldwin",
								"age":     "30",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test histogram with KeyLabel",
			prometheusResponse: promHistogramKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_midichlorians": Metric("metrics.one.midichlorians"),
				},
				Labels: map[string]LabelMap{
					"rank":  KeyLabel("metrics.one.midichlorians.rank"),
					"alive": KeyLabel("metrics.one.midichlorians.alive"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"midichlorians": mapstr.M{
								"count": uint64(86),
								"sum":   1000001.0,
								"bucket": mapstr.M{
									"2000":  uint64(52),
									"4000":  uint64(70),
									"8000":  uint64(78),
									"16000": uint64(84),
									"32000": uint64(86),
									"+Inf":  uint64(86),
								},

								"rank":  "youngling",
								"alive": "yes",
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"midichlorians": mapstr.M{
								"count": uint64(28),
								"sum":   800001.0,
								"bucket": mapstr.M{
									"2000":  uint64(16),
									"4000":  uint64(20),
									"8000":  uint64(23),
									"16000": uint64(27),
									"32000": uint64(27),
									"+Inf":  uint64(28),
								},
								"rank":  "padawan",
								"alive": "yes",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test histogram with KeyLabel With NaN Inf",
			prometheusResponse: promHistogramKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_midichlorians": Metric("metrics.one.midichlorians"),
				},
				Labels: map[string]LabelMap{
					"rank":  KeyLabel("metrics.one.midichlorians.rank"),
					"alive": KeyLabel("metrics.one.midichlorians.alive"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"one": mapstr.M{
							"midichlorians": mapstr.M{
								"count": uint64(86),
								"sum":   1000001.0,
								"bucket": mapstr.M{
									"16000": uint64(84),
									"32000": uint64(86),
									"+Inf":  uint64(86),
								},

								"rank":  "youngling",
								"alive": "yes",
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test summary with KeyLabel",
			prometheusResponse: promSummaryKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_force_propagation_ms": Metric("metrics.force.propagation.ms"),
				},
				Labels: map[string]LabelMap{
					"kind": KeyLabel("metrics.force.propagation.ms.labels.kind"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"force": mapstr.M{
							"propagation": mapstr.M{
								"ms": mapstr.M{
									"count": uint64(651),
									"sum":   89.0,
									"percentile": mapstr.M{
										"0":   35.0,
										"25":  22.0,
										"50":  7.0,
										"75":  20.0,
										"100": 30.0,
									},
									"labels": mapstr.M{
										"kind": "jedi",
									},
								},
							},
						},
					},
				},
				mapstr.M{
					"metrics": mapstr.M{
						"force": mapstr.M{
							"propagation": mapstr.M{
								"ms": mapstr.M{
									"count": uint64(711),
									"sum":   112.0,
									"percentile": mapstr.M{
										"0":   30.0,
										"25":  20.0,
										"50":  12.0,
										"75":  21.0,
										"100": 29.0,
									},
									"labels": mapstr.M{
										"kind": "sith",
									},
								},
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test summary with KeyLabel With NaN Inf",
			prometheusResponse: promSummaryKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_force_propagation_ms": Metric("metrics.force.propagation.ms"),
				},
				Labels: map[string]LabelMap{
					"kind": KeyLabel("metrics.force.propagation.ms.labels.kind"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"force": mapstr.M{
							"propagation": mapstr.M{
								"ms": mapstr.M{
									"count": uint64(651),
									"sum":   50.0,
									"percentile": mapstr.M{
										"75":  20.0,
										"100": 30.0,
									},
									"labels": mapstr.M{
										"kind": "jedi",
									},
								},
							},
						},
					},
				},
			},
		},

		{
			testName:           "Test gauge InfoMetrics using ExtendedInfoMetric",
			prometheusResponse: promGaugeLabeled,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_that_inform_labels": ExtendedInfoMetric(Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "metrics.other_labels"}),
					"metrics_that_use_labels":    Metric("metrics.value"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("metrics.label1"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"value":  20.0,
						"label1": "I am 1",
						"other_labels": mapstr.M{
							"label2": "I am 2",
							"label3": "I am 3",
						},
					},
				},
			},
		},

		{
			testName:           "Test gauge InfoMetrics using ExtendedInfoMetric and extra fields",
			prometheusResponse: promGaugeLabeled,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_that_inform_labels": ExtendedInfoMetric(Configuration{
						StoreNonMappedLabels:     true,
						NonMappedLabelsPlacement: "metrics.other_labels",
						ExtraFields: mapstr.M{
							"metrics.extra.field1": "extra1",
							"metrics.extra.field2": "extra2",
						}}),
					"metrics_that_use_labels": Metric("metrics.value"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("metrics.label1"),
				},
			},
			expectedEvents: []mapstr.M{
				mapstr.M{
					"metrics": mapstr.M{
						"value":  20.0,
						"label1": "I am 1",
						"other_labels": mapstr.M{
							"label2": "I am 2",
							"label3": "I am 3",
						},
						"extra": mapstr.M{
							"field1": "extra1",
							"field2": "extra2",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		r := &mbtest.CapturingReporterV2{}
		p := &prometheus{mockFetcher{response: tc.prometheusResponse}, logp.NewLogger("test")}
		_ = p.ReportProcessedMetrics(tc.mapping, r)
		if !assert.Nil(t, r.GetErrors(),
			"error reporting/processing metrics, at %q", tc.testName) {
			continue
		}

		events := r.GetEvents()
		if !assert.Equal(t, len(tc.expectedEvents), len(events),
			"number of returned events doesn't match expected, at %q", tc.testName) {
			continue
		}

		// Sort slices of received and expeected to avoid unmatching
		sort.Slice(events, func(i, j int) bool {
			return events[i].MetricSetFields.String() < events[j].MetricSetFields.String()
		})
		sort.Slice(tc.expectedEvents, func(i, j int) bool {
			return tc.expectedEvents[i].String() < tc.expectedEvents[j].String()
		})

		for i := range events {
			if !assert.Equal(t, tc.expectedEvents[i], events[i].MetricSetFields,
				"mismatch at event #%d, at %q", i, tc.testName) {

				continue
			}
			t.Logf("events: %+v", events[i].MetricSetFields)
		}
	}
}
