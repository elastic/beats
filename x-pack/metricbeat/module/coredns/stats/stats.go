// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package stats

import (
	"github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{

		// base CoreDNS metrics
		"coredns_panic_count_total":       prometheus.Metric("panic.count"),
		"coredns_dns_request_count_total": prometheus.Metric("dns.request.count"),
		"coredns_dns_request_duration_seconds": prometheus.Metric(
			"dns.request.duration.ns",
			prometheus.OpMultiplyBuckets(1000000000)),
		"coredns_dns_request_size_bytes":         prometheus.Metric("dns.request.size.bytes"),
		"coredns_dns_request_do_count_total":     prometheus.Metric("dns.request.do.count"),
		"coredns_dns_request_type_count_total":   prometheus.Metric("dns.request.type.count"),
		"coredns_dns_response_size_bytes":        prometheus.Metric("dns.response.size.bytes"),
		"coredns_dns_response_rcode_count_total": prometheus.Metric("dns.response.rcode.count"),

		// cache plugin metrics (might not be present if cache plugin is not configured)
		"coredns_cache_hits_total":   prometheus.Metric("dns.cache.hits.count"),
		"coredns_cache_misses_total": prometheus.Metric("dns.cache.misses.count"),
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
