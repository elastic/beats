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

package openmetrics

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

const (
	openMetricsTestSamples = `# TYPE first_metric gauge
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
# TYPE gaugehistogram_metric gaugehistogram
gaugehistogram_metric_bucket{le="0.01"} 20.0
gaugehistogram_metric_bucket{le="0.1"} 25.0
gaugehistogram_metric_bucket{le="1"} 34.0
gaugehistogram_metric_bucket{le="10"} 34.0
gaugehistogram_metric_bucket{le="+Inf"} 42.0
gaugehistogram_metric_gcount 42.0
gaugehistogram_metric_gsum 3289.3
gaugehistogram_metric_created 1520430000.123
# TYPE target info
target_info 1
# TYPE target_with_labels info
target_with_labels_info{env="prod",hostname="myhost"} 1
`

	openMetricsGaugeKeyLabel = `# TYPE metrics_one_count_total gauge
metrics_one_count_total{name="jane",surname="foster"} 1
metrics_one_count_total{name="john",surname="williams"} 2
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3
`

	openMetricsGaugeKeyLabelWithNaNInf = `# TYPE metrics_one_count_errors gauge
metrics_one_count_errors{name="jane",surname="foster"} 0
# TYPE metrics_one_count_total gauge
metrics_one_count_total{name="jane",surname="foster"} NaN
metrics_one_count_total{name="foo",surname="bar"} +Inf
metrics_one_count_total{name="john",surname="williams"} -Inf
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3
`

	openMetricsCounterKeyLabel = `# TYPE metrics_one_count_total counter
metrics_one_count_total{name="jane",surname="foster"} 1
metrics_one_count_total{name="john",surname="williams"} 2
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3
`

	openMetricsCounterKeyLabelWithNaNInf = `# TYPE metrics_one_count_errors counter
metrics_one_count_errors{name="jane",surname="foster"} 1
# TYPE metrics_one_count_total counter
metrics_one_count_total{name="jane",surname="foster"} NaN
metrics_one_count_total{name="john",surname="williams"} +Inf
metrics_one_count_total{name="jahn",surname="baldwin",age="30"} 3

`

	openMetricsHistogramKeyLabel = `# TYPE metrics_one_midichlorians histogram
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

	openMetricsHistogramKeyLabelWithNaNInf = `# TYPE metrics_one_midichlorians histogram
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="2000"} NaN
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="4000"} +Inf
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="8000"} -Inf
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="16000"} 84
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="32000"} 86
metrics_one_midichlorians_bucket{rank="youngling",alive="yes",le="+Inf"} 86
metrics_one_midichlorians_sum{rank="youngling",alive="yes"} 1000001
metrics_one_midichlorians_count{rank="youngling",alive="yes"} 86
`

	openMetricsSummaryKeyLabel = `# TYPE metrics_force_propagation_ms summary
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

	openMetricsSummaryKeyLabelWithNaNInf = `# TYPE metrics_force_propagation_ms summary
metrics_force_propagation_ms{kind="jedi",quantile="0"} NaN
metrics_force_propagation_ms{kind="jedi",quantile="0.25"} +Inf
metrics_force_propagation_ms{kind="jedi",quantile="0.5"} -Inf
metrics_force_propagation_ms{kind="jedi",quantile="0.75"} 20
metrics_force_propagation_ms{kind="jedi",quantile="1"} 30
metrics_force_propagation_ms_sum{kind="jedi"} 50
metrics_force_propagation_ms_count{kind="jedi"} 651
`

	openMetricsGaugeLabeled = `# TYPE metrics_that_inform_labels gauge
metrics_that_inform_labels{label1="I am 1",label2="I am 2"} 1
metrics_that_inform_labels{label1="I am 1",label3="I am 3"} 1
# TYPE metrics_that_use_labels gauge
metrics_that_use_labels{label1="I am 1"} 20
`
	openMetricsStateset = `# TYPE enable_category stateset
enable_category{category="shoes"} 0
enable_category{category="collectibles"} 1
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
	writer.Write([]byte(m.response))
	writer.Close()

	return &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
			"Content-Type":     []string{"application/openmetrics-text"},
		},
		Body: ioutil.NopCloser(body),
	}, nil
}

func TestOpenMetrics(t *testing.T) {

	p := &openmetrics{mockFetcher{response: openMetricsTestSamples}, logp.NewLogger("test")}

	tests := []struct {
		mapping  *MetricsMapping
		msg      string
		expected []common.MapStr
	}{
		{
			msg: "Simple field map",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"first_metric": Metric("first.metric"),
				},
			},
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": 1.0,
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": 1.0,
					},
					"labels": common.MapStr{
						"label3": "Value3",
					},
				},
				common.MapStr{
					"second": common.MapStr{
						"metric": 0.0,
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": 1.0,
					},
					"second": common.MapStr{
						"metric": 0.0,
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": "works",
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": true,
					},
					"second": common.MapStr{
						"metric": false,
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": "Value3",
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": "foo",
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"first": common.MapStr{
						"metric": common.MapStr{
							"foo": "FOO",
						},
					},
					"labels": common.MapStr{
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
			expected: []common.MapStr{},
		},
		{
			msg: "Summary metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"summary_metric": Metric("summary.metric"),
				},
			},
			expected: []common.MapStr{
				common.MapStr{
					"summary": common.MapStr{
						"metric": common.MapStr{
							"sum":   234892394.0,
							"count": uint64(44000),
							"percentile": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"histogram": common.MapStr{
						"metric": common.MapStr{
							"count": uint64(1),
							"bucket": common.MapStr{
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
			expected: []common.MapStr{
				common.MapStr{
					"histogram": common.MapStr{
						"metric": common.MapStr{
							"count": uint64(5),
							"bucket": common.MapStr{
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
		{
			msg: "Gauge histogram metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"gaugehistogram_metric": Metric("gaugehistogram.metric"),
				},
			},
			expected: []common.MapStr{
				common.MapStr{
					"gaugehistogram": common.MapStr{
						"metric": common.MapStr{
							"gcount": uint64(42),
							"bucket": common.MapStr{
								"0.01": uint64(20),
								"0.1":  uint64(25),
								"1":    uint64(34),
								"10":   uint64(34),
								"+Inf": uint64(42),
							},
							"gsum": 3289.3,
						},
					},
				},
			},
		},
		{
			msg: "Info metric",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"target_info": Metric("target_info.metric"),
				},
			},
			expected: []common.MapStr{
				common.MapStr{
					"target_info": common.MapStr{
						"metric": int64(1),
					},
				},
			},
		},
		{
			msg: "Info metric with labels",
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"target_with_labels_info": Metric("target_with_labels_info.metric"),
				},
				Labels: map[string]LabelMap{
					"env":      Label("labels.env"),
					"hostname": Label("labels.hostname"),
				},
			},
			expected: []common.MapStr{
				common.MapStr{
					"target_with_labels_info": common.MapStr{
						"metric": int64(1),
					},
					"labels": common.MapStr{
						"env":      "prod",
						"hostname": "myhost",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			reporter := &mbtest.CapturingReporterV2{}
			p.ReportProcessedMetrics(test.mapping, reporter)
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

func TestOpenMetricsKeyLabels(t *testing.T) {

	testCases := []struct {
		testName            string
		openmetricsResponse string
		mapping             *MetricsMapping
		expectedEvents      []common.MapStr
	}{
		{
			testName:            "Test gauge with KeyLabel",
			openmetricsResponse: openMetricsGaugeKeyLabel,
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
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": 1.0,
							"labels": common.MapStr{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": 2.0,
							"labels": common.MapStr{
								"name":    "john",
								"surname": "williams",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": 3.0,
							"labels": common.MapStr{
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
			testName:            "Test gauge with KeyLabel With NaN Inf",
			openmetricsResponse: openMetricsGaugeKeyLabelWithNaNInf,
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
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": 0.0,
							"labels": common.MapStr{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": 3.0,
							"labels": common.MapStr{
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
			testName:            "Test counter with KeyLabel",
			openmetricsResponse: openMetricsCounterKeyLabel,
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
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": int64(1),
							"labels": common.MapStr{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": int64(2),
							"labels": common.MapStr{
								"name":    "john",
								"surname": "williams",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": int64(3),
							"labels": common.MapStr{
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
			testName:            "Test counter with KeyLabel With NaN Inf",
			openmetricsResponse: openMetricsCounterKeyLabelWithNaNInf,
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
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": int64(1),
							"labels": common.MapStr{
								"name":    "jane",
								"surname": "foster",
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"count": int64(3),
							"labels": common.MapStr{
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
			testName:            "Test histogram with KeyLabel",
			openmetricsResponse: openMetricsHistogramKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_midichlorians": Metric("metrics.one.midichlorians"),
				},
				Labels: map[string]LabelMap{
					"rank":  KeyLabel("metrics.one.midichlorians.rank"),
					"alive": KeyLabel("metrics.one.midichlorians.alive"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"midichlorians": common.MapStr{
								"count": uint64(86),
								"sum":   1000001.0,
								"bucket": common.MapStr{
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
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"midichlorians": common.MapStr{
								"count": uint64(28),
								"sum":   800001.0,
								"bucket": common.MapStr{
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
			testName:            "Test histogram with KeyLabel With NaN Inf",
			openmetricsResponse: openMetricsHistogramKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_one_midichlorians": Metric("metrics.one.midichlorians"),
				},
				Labels: map[string]LabelMap{
					"rank":  KeyLabel("metrics.one.midichlorians.rank"),
					"alive": KeyLabel("metrics.one.midichlorians.alive"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"one": common.MapStr{
							"midichlorians": common.MapStr{
								"count": uint64(86),
								"sum":   1000001.0,
								"bucket": common.MapStr{
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
			testName:            "Test summary with KeyLabel",
			openmetricsResponse: openMetricsSummaryKeyLabel,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_force_propagation_ms": Metric("metrics.force.propagation.ms"),
				},
				Labels: map[string]LabelMap{
					"kind": KeyLabel("metrics.force.propagation.ms.labels.kind"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"force": common.MapStr{
							"propagation": common.MapStr{
								"ms": common.MapStr{
									"count": uint64(651),
									"sum":   89.0,
									"percentile": common.MapStr{
										"0":   35.0,
										"25":  22.0,
										"50":  7.0,
										"75":  20.0,
										"100": 30.0,
									},
									"labels": common.MapStr{
										"kind": "jedi",
									},
								},
							},
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"force": common.MapStr{
							"propagation": common.MapStr{
								"ms": common.MapStr{
									"count": uint64(711),
									"sum":   112.0,
									"percentile": common.MapStr{
										"0":   30.0,
										"25":  20.0,
										"50":  12.0,
										"75":  21.0,
										"100": 29.0,
									},
									"labels": common.MapStr{
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
			testName:            "Test summary with KeyLabel With NaN Inf",
			openmetricsResponse: openMetricsSummaryKeyLabelWithNaNInf,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_force_propagation_ms": Metric("metrics.force.propagation.ms"),
				},
				Labels: map[string]LabelMap{
					"kind": KeyLabel("metrics.force.propagation.ms.labels.kind"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"force": common.MapStr{
							"propagation": common.MapStr{
								"ms": common.MapStr{
									"count": uint64(651),
									"sum":   50.0,
									"percentile": common.MapStr{
										"75":  20.0,
										"100": 30.0,
									},
									"labels": common.MapStr{
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
			testName:            "Test gauge InfoMetrics using ExtendedInfoMetric",
			openmetricsResponse: openMetricsGaugeLabeled,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_that_inform_labels": ExtendedInfoMetric(Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "metrics.other_labels"}),
					"metrics_that_use_labels":    Metric("metrics.value"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("metrics.label1"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"value":  20.0,
						"label1": "I am 1",
						"other_labels": common.MapStr{
							"label2": "I am 2",
							"label3": "I am 3",
						},
					},
				},
			},
		},
		{
			testName:            "Test gauge InfoMetrics using ExtendedInfoMetric and extra fields",
			openmetricsResponse: openMetricsGaugeLabeled,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"metrics_that_inform_labels": ExtendedInfoMetric(Configuration{
						StoreNonMappedLabels:     true,
						NonMappedLabelsPlacement: "metrics.other_labels",
						ExtraFields: common.MapStr{
							"metrics.extra.field1": "extra1",
							"metrics.extra.field2": "extra2",
						}}),
					"metrics_that_use_labels": Metric("metrics.value"),
				},
				Labels: map[string]LabelMap{
					"label1": KeyLabel("metrics.label1"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"value":  20.0,
						"label1": "I am 1",
						"other_labels": common.MapStr{
							"label2": "I am 2",
							"label3": "I am 3",
						},
						"extra": common.MapStr{
							"field1": "extra1",
							"field2": "extra2",
						},
					},
				},
			},
		},
		{
			testName:            "Stateset metric with labels",
			openmetricsResponse: openMetricsStateset,
			mapping: &MetricsMapping{
				Metrics: map[string]MetricMap{
					"enable_category": Metric("metrics.count"),
				},
				Labels: map[string]LabelMap{
					"category": KeyLabel("metrics.labels.category"),
				},
			},
			expectedEvents: []common.MapStr{
				common.MapStr{
					"metrics": common.MapStr{
						"count": int64(0),
						"labels": common.MapStr{
							"category": "shoes",
						},
					},
				},
				common.MapStr{
					"metrics": common.MapStr{
						"count": int64(1),
						"labels": common.MapStr{
							"category": "collectibles",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		r := &mbtest.CapturingReporterV2{}
		p := &openmetrics{mockFetcher{response: tc.openmetricsResponse}, logp.NewLogger("test")}
		p.ReportProcessedMetrics(tc.mapping, r)
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
		}
	}
}
