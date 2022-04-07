// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mesh

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
		"istio_requests_total":           prometheus.Metric("requests"),
		"istio_request_duration_seconds": prometheus.Metric("request.duration.ms", prometheus.OpMultiplyBuckets(1000)),
		"istio_request_bytes":            prometheus.Metric("request.size.bytes"),
		"istio_response_bytes":           prometheus.Metric("response.size.bytes"),
	},

	Labels: map[string]prometheus.LabelMap{
		"instance":                       prometheus.KeyLabel("instance"),
		"job":                            prometheus.KeyLabel("job"),
		"source_workload":                prometheus.KeyLabel("source.workload.name"),
		"source_workload_namespace":      prometheus.KeyLabel("source.workload.namespace"),
		"source_principal":               prometheus.KeyLabel("source.principal"),
		"source_app":                     prometheus.KeyLabel("source.app"),
		"source_version":                 prometheus.KeyLabel("source.version"),
		"destination_workload":           prometheus.KeyLabel("destination.workload.name"),
		"destination_workload_namespace": prometheus.KeyLabel("destination.workload.namespace"),
		"destination_principal":          prometheus.KeyLabel("destination.principal"),
		"destination_app":                prometheus.KeyLabel("destination.app"),
		"destination_version":            prometheus.KeyLabel("destination.version"),
		"destination_service":            prometheus.KeyLabel("destination.service.host"),
		"destination_service_name":       prometheus.KeyLabel("destination.service.name"),
		"destination_service_namespace":  prometheus.KeyLabel("destination.service.namespace"),
		"reporter":                       prometheus.KeyLabel("reporter"),
		"request_protocol":               prometheus.KeyLabel("request.protocol"),
		"response_code":                  prometheus.KeyLabel("response.code"),
		"connection_security_policy":     prometheus.KeyLabel("connection.security.policy"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "mesh",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
