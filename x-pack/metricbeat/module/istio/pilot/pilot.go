// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pilot

import (
	"github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{
		"pilot_xds":                                              prometheus.Metric("xds.count"),
		"pilot_xds_pushes":                                       prometheus.Metric("xds.pushes"),
		"pilot_xds_push_time":                                    prometheus.Metric("xds.push.time.ms", prometheus.OpMultiplyBuckets(1000)),
		"pilot_xds_eds_instances":                                prometheus.Metric("xds.eds.instances"),
		"pilot_xds_push_context_errors":                          prometheus.Metric("xds.push.context.errors"),
		"pilot_total_xds_internal_errors":                        prometheus.Metric("xds.internal.errors"),
		"pilot_conflict_inbound_listener":                        prometheus.Metric("conflict.listener.inbound"),
		"pilot_conflict_outbound_listener_http_over_current_tcp": prometheus.Metric("conflict.listener.outbound.http.over.current.tcp"),
		"pilot_conflict_outbound_listener_http_over_https":       prometheus.Metric("conflict.listener.outbound.http.over.https"),
		"pilot_conflict_outbound_listener_tcp_over_current_http": prometheus.Metric("conflict.listener.outbound.tcp.over.current.http"),
		"pilot_conflict_outbound_listener_tcp_over_current_tcp":  prometheus.Metric("conflict.listener.outbound.tcp.over.current.tcp"),
		"pilot_services":                                         prometheus.Metric("services"),
		"pilot_virt_services":                                    prometheus.Metric("virt.services"),
		"pilot_no_ip":                                            prometheus.Metric("no.ip"),
		"pilot_proxy_convergence_time":                           prometheus.Metric("proxy.conv.ms", prometheus.OpMultiplyBuckets(1000)),
	},

	Labels: map[string]prometheus.LabelMap{
		"cluster": prometheus.KeyLabel("cluster"),
		"type":    prometheus.KeyLabel("type"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "pilot",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
