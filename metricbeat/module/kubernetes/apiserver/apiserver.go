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

package apiserver

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{

			// Process metrics
			"process_cpu_seconds_total":     prometheus.Metric("process.cpu.sec"),
			"process_resident_memory_bytes": prometheus.Metric("process.memory.resident.bytes"),
			"process_virtual_memory_bytes":  prometheus.Metric("process.memory.virtual.bytes"),
			"process_open_fds":              prometheus.Metric("process.fds.open.count"),
			"process_start_time_seconds":    prometheus.Metric("process.started.sec"),

			// HTTP server metrics
			"http_request_duration_microseconds": prometheus.Metric("http.request.duration.us"),
			"http_request_size_bytes":            prometheus.Metric("http.request.size.bytes"),
			"http_response_size_bytes":           prometheus.Metric("http.response.size.bytes"),
			"http_requests_total":                prometheus.Metric("http.request.count"),

			// REST metrics
			"rest_client_requests_total": prometheus.Metric("client.request.count"),

			// Deprecated, remove in future releases
			"apiserver_request_latencies": prometheus.Metric("request.latency"),

			"apiserver_request_total":                 prometheus.Metric("request.count"),
			"apiserver_current_inflight_requests":     prometheus.Metric("request.current.count"),
			"apiserver_longrunning_gauge":             prometheus.Metric("request.longrunning.count"),
			"etcd_object_counts":                      prometheus.Metric("etcd.object.count"),
			"apiserver_audit_event_total":             prometheus.Metric("audit.event.count"),
			"apiserver_audit_requests_rejected_total": prometheus.Metric("audit.rejected.count"),
		},

		Labels: map[string]prometheus.LabelMap{
			"client":      prometheus.KeyLabel("client"),
			"code":        prometheus.KeyLabel("code"),
			"contentType": prometheus.KeyLabel("content_type"),
			"dry_run":     prometheus.KeyLabel("dry_run"),
			"requestKind": prometheus.KeyLabel("kind"),

			"verb":        prometheus.KeyLabel("verb"),
			"scope":       prometheus.KeyLabel("scope"),
			"resource":    prometheus.KeyLabel("resource"),
			"subresource": prometheus.KeyLabel("subresource"),
			"component":   prometheus.KeyLabel("component"),
			"group":       prometheus.KeyLabel("group"),
			"version":     prometheus.KeyLabel("version"),

			"handler": prometheus.KeyLabel("handler"),
			"method":  prometheus.KeyLabel("method"),
			"host":    prometheus.KeyLabel("host"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "apiserver",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
