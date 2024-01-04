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
	"testing"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func stringp(x string) *string {
	return &x
}

func float64p(x float64) *float64 {
	return &x
}

func uint64p(x uint64) *uint64 {
	return &x
}

func int64p(x int64) *int64 {
	return &x
}

func TestCounterOpenMetrics(t *testing.T) {
	input := `
# TYPE process_cpu_total counter
# HELP process_cpu_total Some help.
process_cpu_total 4.20072246e+06
# TYPE something counter
something_total 20
# TYPE metric_without_suffix counter
metric_without_suffix 10
# EOF
`

	expected := []*MetricFamily{
		{
			Name: stringp("process_cpu_total"),
			Help: stringp("Some help."),
			Type: "counter",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("process_cpu_total"),
					Counter: &Counter{
						Value: float64p(4.20072246e+06),
					},
				},
			},
		},
		{
			Name: stringp("something"),
			Help: nil,
			Type: "counter",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("something_total"),
					Counter: &Counter{
						Value: float64p(20),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestCounterPrometheus(t *testing.T) {
	input := `
# TYPE process_cpu_total counter
# HELP process_cpu_total Some help.
process_cpu_total 4.20072246e+06
# TYPE process_cpu counter
process_cpu 20
`

	expected := []*MetricFamily{
		{
			Name: stringp("process_cpu_total"),
			Help: stringp("Some help."),
			Type: "counter",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("process_cpu_total"),
					Counter: &Counter{
						Value: float64p(4.20072246e+06),
					},
				},
			},
		},
		{
			Name: stringp("process_cpu"),
			Help: nil,
			Type: "counter",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("process_cpu"),
					Counter: &Counter{
						Value: float64p(20),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input), ContentTypeTextFormat, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", ContentTypeTextFormat)
	}
	require.ElementsMatch(t, expected, result)
}

func TestGaugeOpenMetrics(t *testing.T) {
	input := `
# TYPE first_metric gauge
first_metric{label1="value1"} 1
# TYPE second_metric gauge
# HELP second_metric Help for gauge metric.
second_metric 0
# EOF
`
	expected := []*MetricFamily{
		{
			Name: stringp("first_metric"),
			Help: nil,
			Type: "gauge",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "label1",
							Value: "value1",
						},
					},
					Name: stringp("first_metric"),
					Gauge: &Gauge{
						Value: float64p(1),
					},
				},
			},
		},
		{
			Name: stringp("second_metric"),
			Help: stringp("Help for gauge metric."),
			Type: "gauge",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("second_metric"),
					Gauge: &Gauge{
						Value: float64p(0),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestGaugePrometheus(t *testing.T) {
	input := `
# TYPE first_metric gauge
first_metric{label1="value1"} 1
# TYPE second_metric gauge
# HELP second_metric Help for gauge metric.
second_metric 0
`
	expected := []*MetricFamily{
		{
			Name: stringp("first_metric"),
			Help: nil,
			Type: "gauge",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "label1",
							Value: "value1",
						},
					},
					Name: stringp("first_metric"),
					Gauge: &Gauge{
						Value: float64p(1),
					},
				},
			},
		},
		{
			Name: stringp("second_metric"),
			Help: stringp("Help for gauge metric."),
			Type: "gauge",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("second_metric"),
					Gauge: &Gauge{
						Value: float64p(0),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), ContentTypeTextFormat, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", ContentTypeTextFormat)
	}
	require.ElementsMatch(t, expected, result)
}

func TestInfoOpenMetrics(t *testing.T) {
	input := `
# TYPE target info
target_info 1
# TYPE metric_info info
metric_info 2
# TYPE metric_without_suffix info
metric_without_suffix 3
# EOF
`
	expected := []*MetricFamily{
		{
			Name: stringp("target"),
			Help: nil,
			Type: "info",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("target_info"),
					Info: &Info{
						Value: int64p(1),
					},
				},
			},
		},
		{
			Name: stringp("metric_info"),
			Help: nil,
			Type: "info",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("metric_info"),
					Info: &Info{
						Value: int64p(2),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestInfoPrometheus(t *testing.T) {
	input := `
# TYPE target info
target_info 1
# TYPE first_metric gauge
first_metric{label1="value1"} 1
# EOF
`
	expected := []*MetricFamily{
		{
			Name: stringp("target_info"),
			Help: nil,
			Type: "unknown",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("target_info"),
					Unknown: &Unknown{
						Value: float64p(1),
					},
				},
			},
		},
		{
			Name: stringp("first_metric"),
			Help: nil,
			Type: "gauge",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "label1",
							Value: "value1",
						},
					},
					Name: stringp("first_metric"),
					Gauge: &Gauge{
						Value: float64p(1),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input), ContentTypeTextFormat, time.Now(), logp.NewLogger("test"))
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestStatesetOpenMetrics(t *testing.T) {
	input := `
# TYPE a stateset
# HELP a help
a{a="bar"} 0
a{a="foo"} 1.0
# EOF
`
	expected := []*MetricFamily{
		{
			Name: stringp("a"),
			Help: stringp("help"),
			Type: "stateset",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "a",
							Value: "bar",
						},
					},
					Name: stringp("a"),
					Stateset: &Stateset{
						Value: int64p(0),
					},
				},
				{
					Label: []*labels.Label{
						{
							Name:  "a",
							Value: "foo",
						},
					},
					Name: stringp("a"),
					Stateset: &Stateset{
						Value: int64p(1),
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestSummaryOpenMetrics(t *testing.T) {
	input := `
# TYPE summary_metric summary
summary_metric{quantile="0.5"} 29735
summary_metric{quantile="0.9"} 47103
summary_metric{quantile="0.99"} 50681
summary_metric{noquantile="0.2"} 50681
summary_metric_sum 234892394
summary_metric_count 44000
summary_metric_impossible 123
# EOF
`
	expected := []*MetricFamily{
		{
			Name: stringp("summary_metric"),
			Help: nil,
			Type: "summary",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("summary_metric"),
					Summary: &Summary{
						SampleCount: uint64p(44000),
						SampleSum:   float64p(234892394),
						Quantile: []*Quantile{
							{
								Quantile: float64p(0.5),
								Value:    float64p(29735),
							},
							{
								Quantile: float64p(0.9),
								Value:    float64p(47103),
							},
							{
								Quantile: float64p(0.99),
								Value:    float64p(50681),
							},
						},
					},
					TimestampMs: nil,
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestSummaryPrometheus(t *testing.T) {
	input := `
# TYPE summary_metric summary
summary_metric{quantile="0.5"} 29735
summary_metric{quantile="0.9"} 47103
summary_metric{quantile="0.99"} 50681
summary_metric{noquantile="0.2"} 50681
summary_metric_sum 234892394
summary_metric_count 44000
summary_metric_impossible 123
`
	expected := []*MetricFamily{
		{
			Name: stringp("summary_metric"),
			Help: nil,
			Type: "summary",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("summary_metric"),
					Summary: &Summary{
						SampleCount: uint64p(44000),
						SampleSum:   float64p(234892394),
						Quantile: []*Quantile{
							{
								Quantile: float64p(0.5),
								Value:    float64p(29735),
							},
							{
								Quantile: float64p(0.9),
								Value:    float64p(47103),
							},
							{
								Quantile: float64p(0.99),
								Value:    float64p(50681),
							},
						},
					},
					TimestampMs: nil,
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input), ContentTypeTextFormat, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", ContentTypeTextFormat)
	}
	require.ElementsMatch(t, expected, result)
}

func TestHistogramOpenMetrics(t *testing.T) {
	input := `
# HELP http_server_requests_seconds Duration of HTTP server request handling
# TYPE http_server_requests_seconds histogram
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",quantile="0.002796201"} 0.046137344
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",quantile="0.003145726"} 0.046137344
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",noquantile="0.005"} 1234
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",le="0.001"} 0.0
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",le="0.001048576"} 0.0
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",nole="0.002"} 1234
http_server_requests_seconds_count{exception="None",uri="/actuator/prometheus"} 1.0
http_server_requests_seconds_sum{exception="None",uri="/actuator/prometheus"} 0.046745444
http_server_requests_seconds_created{exception="None",uri="/actuator/prometheus"} 0.046745444
# EOF`
	expected := []*MetricFamily{
		{
			Name: stringp("http_server_requests_seconds"),
			Help: stringp("Duration of HTTP server request handling"),
			Type: "histogram",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "exception",
							Value: "None",
						},
						{
							Name:  "uri",
							Value: "/actuator/prometheus",
						},
					},
					Name: stringp("http_server_requests_seconds"),
					Histogram: &Histogram{
						IsGaugeHistogram: false,
						SampleCount:      uint64p(1.0),
						SampleSum:        float64p(0.046745444),
						Bucket: []*Bucket{
							{
								CumulativeCount: uint64p(0),
								UpperBound:      float64p(0.001),
							},
							{
								CumulativeCount: uint64p(0),
								UpperBound:      float64p(0.001048576),
							},
						},
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestHistogramPrometheus(t *testing.T) {
	input := `
# HELP http_server_requests_seconds Duration of HTTP server request handling
# TYPE http_server_requests_seconds histogram
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",quantile="0.002796201"} 0.046137344
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",quantile="0.003145726"} 0.046137344
http_server_requests_seconds{exception="None",uri="/actuator/prometheus",noquantile="0.005"} 1234
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",le="0.001"} 0.0
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",le="0.001048576"} 0.0
http_server_requests_seconds_bucket{exception="None",uri="/actuator/prometheus",nole="0.002"} 1234
http_server_requests_seconds_count{exception="None",uri="/actuator/prometheus"} 1.0
http_server_requests_seconds_sum{exception="None",uri="/actuator/prometheus"} 0.046745444
http_server_requests_seconds_created{exception="None",uri="/actuator/prometheus"} 0.046745444`
	expected := []*MetricFamily{
		{
			Name: stringp("http_server_requests_seconds"),
			Help: stringp("Duration of HTTP server request handling"),
			Type: "histogram",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "exception",
							Value: "None",
						},
						{
							Name:  "uri",
							Value: "/actuator/prometheus",
						},
					},
					Name: stringp("http_server_requests_seconds"),
					Histogram: &Histogram{
						IsGaugeHistogram: false,
						SampleCount:      uint64p(1.0),
						SampleSum:        float64p(0.046745444),
						Bucket: []*Bucket{
							{
								CumulativeCount: uint64p(0),
								UpperBound:      float64p(0.001),
							},
							{
								CumulativeCount: uint64p(0),
								UpperBound:      float64p(0.001048576),
							},
						},
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input), ContentTypeTextFormat, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", ContentTypeTextFormat)
	}
	require.ElementsMatch(t, expected, result)
}

func TestGaugeHistogramOpenMetrics(t *testing.T) {
	input := `
# TYPE ggh gaugehistogram
ggh_bucket{le=".9"} 2
ggh_bucket{nole=".99"} 10
ggh_gcount 2
ggh_gsum 1
ggh_impossible 321
ggh 99
# EOF`
	expected := []*MetricFamily{
		{
			Name: stringp("ggh"),
			Help: nil,
			Type: "gaugehistogram",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{},
					Name:  stringp("ggh"),
					Histogram: &Histogram{
						IsGaugeHistogram: true,
						SampleCount:      uint64p(2.0),
						SampleSum:        float64p(1),
						Bucket: []*Bucket{
							{
								CumulativeCount: uint64p(2),
								UpperBound:      float64p(0.9),
							},
						},
					},
				},
			},
		},
	}

	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestUnknownOpenMetrics(t *testing.T) {
	input := `
# HELP redis_connected_clients Redis connected clients
# TYPE redis_connected_clients unknown
redis_connected_clients{instance="rough-snowflake-web"} 10.0
# EOF`
	expected := []*MetricFamily{
		{
			Name: stringp("redis_connected_clients"),
			Help: stringp("Redis connected clients"),
			Type: "unknown",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "instance",
							Value: "rough-snowflake-web",
						},
					},
					Name: stringp("redis_connected_clients"),
					Unknown: &Unknown{
						Value: float64p(10),
					},
				},
			},
		},
	}
	result, err := ParseMetricFamilies([]byte(input[1:]), OpenMetricsType, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", OpenMetricsType)
	}
	require.ElementsMatch(t, expected, result)
}

func TestUntypedPrometheus(t *testing.T) {
	input := `
# HELP redis_connected_clients Redis connected clients
# TYPE redis_connected_clients untyped
redis_connected_clients{instance="rough-snowflake-web"} 10.0`
	expected := []*MetricFamily{
		{
			Name: stringp("redis_connected_clients"),
			Help: stringp("Redis connected clients"),
			Type: "unknown",
			Unit: nil,
			Metric: []*OpenMetric{
				{
					Label: []*labels.Label{
						{
							Name:  "instance",
							Value: "rough-snowflake-web",
						},
					},
					Name: stringp("redis_connected_clients"),
					Unknown: &Unknown{
						Value: float64p(10),
					},
				},
			},
		},
	}
	result, err := ParseMetricFamilies([]byte(input), ContentTypeTextFormat, time.Now(), nil)
	if err != nil {
		t.Fatalf("ParseMetricFamilies for content type %s returned an error.", ContentTypeTextFormat)
	}
	require.ElementsMatch(t, expected, result)
}
