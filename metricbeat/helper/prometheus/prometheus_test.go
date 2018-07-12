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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

const promMetrics = `
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

`

type mockFetcher struct{}

func (m mockFetcher) FetchResponse() (*http.Response, error) {
	return &http.Response{
		Header: make(http.Header),
		Body:   ioutil.NopCloser(bytes.NewReader([]byte(promMetrics))),
	}, nil
}

func TestPrometheus(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "text/plain; charset=ISO-8859-1")
		w.Write([]byte(promMetrics))
	}))

	server.Start()
	defer server.Close()

	p := &prometheus{mockFetcher{}}

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
					"first_metric": LabelMetric("first.metric", "label3", false),
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
					"first_metric": LabelMetric("first.metric", "label4", true),
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
	}

	for _, test := range tests {
		reporter := &mbtest.CapturingReporterV2{}
		p.ReportProcessedMetrics(test.mapping, reporter)
		assert.Nil(t, reporter.GetErrors(), test.msg)
		// Sort slice to avoid randomness
		res := reporter.GetEvents()
		sort.Slice(res, func(i, j int) bool {
			return res[i].MetricSetFields.String() < res[j].MetricSetFields.String()
		})
		for j, ev := range res {
			assert.Equal(t, test.expected[j], ev.MetricSetFields, test.msg)
		}
	}
}
