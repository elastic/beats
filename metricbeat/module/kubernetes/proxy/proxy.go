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

package proxy

import (
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{

			"process_cpu_seconds_total":          prometheus.Metric("process.cpu.sec"),
			"process_resident_memory_bytes":      prometheus.Metric("process.memory.resident.bytes"),
			"process_virtual_memory_bytes":       prometheus.Metric("process.memory.virtual.bytes"),
			"process_open_fds":                   prometheus.Metric("process.fds.open.count"),
			"process_start_time_seconds":         prometheus.Metric("process.started.sec"),
			"http_request_duration_microseconds": prometheus.Metric("http.request.duration.us"),
			"http_request_size_bytes":            prometheus.Metric("http.request.size.bytes"),
			"http_response_size_bytes":           prometheus.Metric("http.response.size.bytes"),
			"http_requests_total":                prometheus.Metric("http.request.count"),
			"rest_client_requests_total":         prometheus.Metric("client.request.count"),
			"kubeproxy_sync_proxy_rules_duration_seconds": prometheus.Metric("sync.rules.duration.us",
				prometheus.OpMultiplyBuckets(1000000)),
			"kubeproxy_network_programming_duration_seconds": prometheus.Metric("sync.networkprogramming.duration.us",
				prometheus.OpMultiplyBuckets(1000000)),
		},

		Labels: map[string]prometheus.LabelMap{
			"code":    prometheus.KeyLabel("code"),
			"host":    prometheus.KeyLabel("host"),
			"method":  prometheus.KeyLabel("method"),
			"handler": prometheus.KeyLabel("handler"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "proxy",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
