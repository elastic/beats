// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mesh

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
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
		"istio_requests_total":           prometheus.Metric("requests"),
		"istio_request_duration_seconds": prometheus.Metric("request.duration.count"),
		"istio_request_bytes":            prometheus.Metric("request.bytes.count"),
		"istio_response_bytes":           prometheus.Metric("response.bytes.count"),
	},

	Labels: map[string]prometheus.LabelMap{
		"instance":                      prometheus.KeyLabel("instance"),
		"job":                           prometheus.KeyLabel("job"),
		"connection_security_policy":    prometheus.KeyLabel("connection_security_policy"),
		"destination_app":               prometheus.KeyLabel("destination.app"),
		"destination_service":           prometheus.KeyLabel("destination.service.path"),
		"destination_service_name":      prometheus.KeyLabel("destination.service.name"),
		"destination_service_namespace": prometheus.KeyLabel("destination.service.namespace"),
		"destination_version":           prometheus.KeyLabel("destination.version"),

		"destination_workload_namespace": prometheus.KeyLabel("destination.workload_namespace"),
		"reporter":                       prometheus.KeyLabel("reporter"),
		"request_protocol":               prometheus.KeyLabel("request.protocol"),
		"response_code":                  prometheus.KeyLabel("response.code"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "mesh",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
