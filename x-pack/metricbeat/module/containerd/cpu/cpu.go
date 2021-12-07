// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cpu

import (
	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/v1/metrics"
)

var (
	// HostParser validates Prometheus URLs
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	// Mapping of state metrics
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"container_cpu_total_nanoseconds":  prometheus.Metric("usage.total.ns"),
			"container_cpu_user_nanoseconds":   prometheus.Metric("usage.user.ns"),
			"container_cpu_kernel_nanoseconds": prometheus.Metric("usage.kernel.ns"),
			"container_per_cpu_nanoseconds":    prometheus.Metric("usage.percpu.ns"),
			"process_cpu_seconds_total":        prometheus.Metric("system.total"),
		},
		Labels: map[string]prometheus.LabelMap{
			"container_id": prometheus.KeyLabel("id"),
			"cpu":          prometheus.KeyLabel("cpu"),
		},
	}

	mb.Registry.MustAddMetricSet("containerd", "cpu",
		getMetricsetFactory(mapping),
		mb.WithHostParser(hostParser),
	)
}
