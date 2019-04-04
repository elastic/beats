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

package stats

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{
		"coredns_panic_count_total":       prometheus.Metric("panic.count.total"),
		"coredns_dns_request_count_total": prometheus.Metric("dns.request.count.total"),
		"coredns_dns_request_duration_seconds": prometheus.Metric(
			"dns.request.duration.ns",
			prometheus.OpMultiplyBuckets(1000000000)),
		"coredns_dns_request_size_bytes":         prometheus.Metric("dns.request.size.bytes"),
		"coredns_dns_request_do_count_total":     prometheus.Metric("dns.request.do.count.total"),
		"coredns_dns_request_type_count_total":   prometheus.Metric("dns.request.type.count.total"),
		"coredns_dns_response_size_bytes":        prometheus.Metric("dns.response.size.bytes"),
		"coredns_dns_response_rcode_count_total": prometheus.Metric("dns.response.rcode.count.total"),
	},

	Labels: map[string]prometheus.LabelMap{
		"server": prometheus.KeyLabel("server"),
		"zone":   prometheus.KeyLabel("zone"),
		"type":   prometheus.KeyLabel("type"),
		"rcode":  prometheus.KeyLabel("rcode"),
		"proto":  prometheus.KeyLabel("proto"),
		"family": prometheus.KeyLabel("family"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("coredns", "stats",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
