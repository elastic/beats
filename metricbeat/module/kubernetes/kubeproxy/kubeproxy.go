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

package kubeproxy

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"process_cpu_seconds_total":     prometheus.Metric("process.cpu.sec"),
			"process_resident_memory_bytes": prometheus.Metric("process.memory.resident.bytes"),
			"process_virtual_memory_bytes":  prometheus.Metric("process.memory.virtual.bytes"),

			"kubeproxy_sync_proxy_rules_latency_microseconds_bucket": prometheus.Metric("sync.rules.latency.us"),

			"rest_client_request_latency_seconds_bucket": prometheus.Metric("client.request.latency.us",
				prometheus.OpMultiplyBuckets(1000000)),
			"rest_client_requests_total": prometheus.Metric("client.request.count"),
		},

		Labels: map[string]prometheus.LabelMap{
			"code": prometheus.KeyLabel("client.request.status_code"),
			"host": prometheus.KeyLabel("client.request.host"),
			// TODO method and verb are used to store the same info at different buckets
			// We need to make sure this doesn't overwrites data at the prometheus helper
			"method": prometheus.KeyLabel("client.request.method"),
			"verb":   prometheus.KeyLabel("client.request.method"),
			"url":    prometheus.KeyLabel("client.request.url"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "kubeproxy",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
