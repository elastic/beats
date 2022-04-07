// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package citadel

import (
	"github.com/elastic/beats/v8/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
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
		"citadel_secret_controller_svc_acc_created_cert_count": prometheus.Metric("secret_controller_svc_acc_created_cert.count"),
		"citadel_server_root_cert_expiry_timestamp":            prometheus.Metric("server_root_cert_expiry_seconds"),
		"grpc_server_handled_total":                            prometheus.Metric("grpc.server.handled"),
		"grpc_server_handling_seconds":                         prometheus.Metric("grpc.server.handling.latency.ms", prometheus.OpMultiplyBuckets(1000)),
		"grpc_server_msg_received_total":                       prometheus.Metric("grpc.server.msg.received"),
		"grpc_server_msg_sent_total":                           prometheus.Metric("grpc.server.msg.sent"),
		"grpc_server_started_total":                            prometheus.Metric("grpc.server.started"),
	},

	Labels: map[string]prometheus.LabelMap{
		"grpc_method":  prometheus.KeyLabel("grpc.method"),
		"grpc_service": prometheus.KeyLabel("grpc.service"),
		"grpc_type":    prometheus.KeyLabel("grpc.type"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "citadel",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
