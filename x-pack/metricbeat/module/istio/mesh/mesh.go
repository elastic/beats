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
	defaultPath   = "/federate"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		QueryParams: mb.QueryParams{"match[]": "{__name__=~\"istio.*\"}"}.String(),
	}.Build()
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{

		// base CoreDNS metrics
		"istio_requests_total": prometheus.Metric("requests", ),
	},

	Labels: map[string]prometheus.LabelMap{
		"instance": prometheus.KeyLabel("instance"),
		"job":      prometheus.KeyLabel("job"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "mesh",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
