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

	"github.com/elastic/elastic-agent-libs/logp"
)

func FuzzParseMetricFamilies(f *testing.F) {
	seeds := [][]byte{
		// Valid metrics
		[]byte("# TYPE http_requests counter\nhttp_requests 100\n"),
		[]byte("# TYPE http_requests_total counter\nhttp_requests_total 100\n"),
		[]byte("# TYPE temperature gauge\ntemperature 23.5\n"),
		[]byte("# TYPE temperature gauge\ntemperature{location=\"room1\"} 23.5\n"),
		[]byte(`# TYPE http_duration histogram
http_duration_bucket{le="0.1"} 10
http_duration_bucket{le="0.5"} 50
http_duration_bucket{le="+Inf"} 100
http_duration_sum 35.5
http_duration_count 100
`),
		[]byte(`# TYPE rpc_duration summary
rpc_duration{quantile="0.5"} 0.05
rpc_duration{quantile="0.9"} 0.08
rpc_duration{quantile="0.99"} 0.1
rpc_duration_sum 17.5
rpc_duration_count 200
`),
		[]byte("# TYPE custom_metric unknown\ncustom_metric 42\n"),
		[]byte("metric_without_type 42\n"),

		// OpenMetrics
		[]byte("# TYPE http_requests counter\nhttp_requests_total 100\nhttp_requests_created 1234567890\n# EOF\n"),
		[]byte("# TYPE build info\nbuild_info{version=\"1.0\",commit=\"abc123\"} 1\n# EOF\n"),
		[]byte("# TYPE feature stateset\nfeature{feature=\"a\"} 1\nfeature{feature=\"b\"} 0\n# EOF\n"),
		[]byte(`# TYPE request_size gaugehistogram
request_size_gcount 100
request_size_gsum 12345
request_size_bucket{le="100"} 10
request_size_bucket{le="+Inf"} 100
# EOF
`),

		// Exemplars
		[]byte("# TYPE http_requests counter\nhttp_requests_total 100 # {trace_id=\"abc\"} 1.0\n# EOF\n"),
		[]byte("# TYPE http_requests counter\nhttp_requests_total 100 # {trace_id=\"abc\",span_id=\"def\"} 1.0 123456\n# EOF\n"),
		[]byte("# TYPE http_duration histogram\nhttp_duration_bucket{le=\"1\"} 10 # {trace_id=\"xyz\"} 0.9\n# EOF\n"),

		// Metadata
		[]byte("# HELP http_requests Total HTTP requests\n# TYPE http_requests counter\nhttp_requests 100\n"),
		[]byte("# TYPE temperature gauge\n# UNIT temperature celsius\ntemperature 23.5\n# EOF\n"),

		// Timestamps and labels
		[]byte("metric_with_ts 100 1234567890\n"),
		[]byte("metric_with_ts{label=\"value\"} 100 1234567890\n"),
		[]byte("metric{a=\"1\",b=\"2\",c=\"3\",d=\"4\"} 100\n"),

		// Special values
		[]byte("metric_nan NaN\n"),
		[]byte("metric_inf +Inf\n"),
		[]byte("metric_neginf -Inf\n"),

		// Edge cases
		nil,
		{},
		[]byte("\n"),

		// Malformed labels
		[]byte("metric{"),
		[]byte("metric{label}"),
		[]byte("metric{label=}"),
		[]byte("metric{label=\""),

		// Known crash inputs
		[]byte("{A}0"),
		[]byte("{A}00"),
		[]byte("{A}000"),
		[]byte("{A}0000"),
		[]byte("{A} 1"),
		[]byte("{A} 1\n"),
		[]byte("{A}0\n"),
		[]byte("{A}0 1"),
		[]byte("{A}0 1\n"),
		[]byte("{A}0\n000"),
		[]byte("{A}00\n"),
		[]byte("{A}00000"),
		[]byte("{A}00 1"),
		[]byte("{A}00 1\n"),
		[]byte("{A}00\n000"),

		// Malformed exemplars
		[]byte("# TYPE c counter\nc_total 10 # {}\n# EOF\n"),
		[]byte("# TYPE c counter\nc_total 10 # {\n# EOF\n"),
		[]byte("# TYPE c counter\nc_total 10 # {a=}\n# EOF\n"),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	logger := logp.NewLogger("fuzz")

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseMetricFamilies(data, ContentTypeTextFormat, time.Now(), logger)
		_, _ = ParseMetricFamilies(data, OpenMetricsType, time.Now(), logger)
		_, _ = ParseMetricFamilies(data, "", time.Now(), logger)
	})
}
