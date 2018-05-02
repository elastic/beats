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
)

const promMetrics = `
# TYPE first_metric gauge
first_metric{label1="value1",label2="value2",label3="value3"} 1
# TYPE second_metric gauge
second_metric{label1="value1",label3="othervalue"} 0
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
					"first.metric": 1.0,
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
					"first.metric":  1.0,
					"labels.label1": "value1",
					"labels.label2": "value2",
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
					"first.metric":  1.0,
					"labels.label3": "value3",
				},
				common.MapStr{
					"second.metric": 0.0,
					"labels.label3": "othervalue",
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
					"first.metric":  1.0,
					"second.metric": 0.0,
					"labels.label1": "value1",
					"labels.label2": "value2",
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
					"first.metric":  "works",
					"labels.label1": "value1",
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
					"first.metric":  true,
					"second.metric": false,
					"labels.label1": "value1",
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
					"first.metric":  "value3",
					"labels.label1": "value1",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := p.GetProcessedMetrics(test.mapping)
		assert.Nil(t, err, test.msg)
		// Sort slice to avoid randomness
		sort.Slice(res, func(i, j int) bool {
			return res[i].String() < res[j].String()
		})
		assert.Equal(t, test.expected, res, test.msg)
	}
}
