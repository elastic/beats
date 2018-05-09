package apiserver

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"apiserver_request_count":             prometheus.Metric("request.count"),
			"apiserver_request_latencies_summary": prometheus.Metric("request.latency"),
		},

		Labels: map[string]prometheus.LabelMap{
			"client":      prometheus.KeyLabel("client"),
			"resource":    prometheus.KeyLabel("resource"),
			"scope":       prometheus.KeyLabel("scope"),
			"subresource": prometheus.KeyLabel("subresource"),
			"verb":        prometheus.KeyLabel("verb"),
		},
	}

	if err := mb.Registry.AddMetricSet("kubernetes", "apiserver",
		prometheus.MetricSetBuilder(mapping), prometheus.HostParser); err != nil {
		panic(err)
	}
}
