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
	"github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

func init() {
	//mapping := &prometheus.MetricsMapping{
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"process_cpu_seconds_total":               prometheus.Metric("process.cpu.sec"),
			"process_resident_memory_bytes":           prometheus.Metric("process.memory.resident.bytes"),
			"process_virtual_memory_bytes":            prometheus.Metric("process.memory.virtual.bytes"),
			"process_open_fds":                        prometheus.Metric("process.fds.open.count"),
			"process_start_time_seconds":              prometheus.Metric("process.started.sec"),
			"http_request_duration_microseconds":      prometheus.Metric("http.request.duration.us"),
			"http_request_size_bytes":                 prometheus.Metric("http.request.size.bytes"),
			"http_response_size_bytes":                prometheus.Metric("http.response.size.bytes"),
			"http_requests_total":                     prometheus.Metric("http.request.count"),
			"rest_client_requests_total":              prometheus.Metric("client.request.count"),
			"apiserver_request_duration_seconds":      prometheus.Metric("request.duration.us", prometheus.OpMultiplyBuckets(1000000)),
			"apiserver_request_latencies":             prometheus.Metric("request.latency"),
			"apiserver_request_total":                 prometheus.Metric("request.count"),
			"apiserver_request_count":                 prometheus.Metric("request.beforev14.count"),
			"apiserver_current_inflight_requests":     prometheus.Metric("request.current.count"),
			"apiserver_longrunning_gauge":             prometheus.Metric("request.longrunning.count"),
			"etcd_object_counts":                      prometheus.Metric("etcd.object.count"),
			"apiserver_audit_event_total":             prometheus.Metric("audit.event.count"),
			"apiserver_audit_requests_rejected_total": prometheus.Metric("audit.rejected.count"),
		},

		Labels: map[string]prometheus.LabelMap{
			"client":      prometheus.KeyLabel("request.client"),
			"resource":    prometheus.KeyLabel("request.resource"),
			"scope":       prometheus.KeyLabel("request.scope"),
			"subresource": prometheus.KeyLabel("request.subresource"),
			"verb":        prometheus.KeyLabel("request.verb"),
			"code":        prometheus.KeyLabel("request.code"),
			"contentType": prometheus.KeyLabel("request.content_type"),
			"dry_run":     prometheus.KeyLabel("request.dry_run"),
			"requestKind": prometheus.KeyLabel("request.kind"),
			"component":   prometheus.KeyLabel("request.component"),
			"group":       prometheus.KeyLabel("request.group"),
			"version":     prometheus.KeyLabel("request.version"),
			"handler":     prometheus.KeyLabel("request.handler"),
			"method":      prometheus.KeyLabel("request.method"),
			"host":        prometheus.KeyLabel("request.host"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "apiserver",
		getMetricsetFactory(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
