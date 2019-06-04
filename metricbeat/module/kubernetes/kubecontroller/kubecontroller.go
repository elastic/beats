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

package kubecontroller

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			// HTTP server metrics
			"http_request_duration_microseconds": prometheus.Metric("http.request.duration.us"),
			"http_request_size_bytes":            prometheus.Metric("http.request.bytes"),
			"http_response_size_bytes":           prometheus.Metric("http.response.bytes"),
			"http_requests_total":                prometheus.Metric("http.response.count"),

			// REST metrics
			"rest_client_request_duration_seconds": prometheus.Metric("client.request.duration.us",
				prometheus.OpMultiplyBuckets(1000000)),
			"rest_client_requests_total": prometheus.Metric("client.response.count"),
		},

		Labels: map[string]prometheus.LabelMap{
			"handler": prometheus.KeyLabel("handler"),
			"code":    prometheus.KeyLabel("code"),
			"url":     prometheus.KeyLabel("url"),
			"verb":    prometheus.KeyLabel("verb"),
			"method":  prometheus.KeyLabel("method"),
			"host":    prometheus.KeyLabel("host"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "kubecontroller",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
