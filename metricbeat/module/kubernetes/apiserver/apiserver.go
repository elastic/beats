package apiserver

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"apiserver_request_count":     prometheus.Metric("request.count"),
			"apiserver_request_latencies": prometheus.Metric("request.latency"),
		},

		Labels: map[string]prometheus.LabelMap{
			"client":      prometheus.KeyLabel("request.client"),
			"resource":    prometheus.KeyLabel("request.resource"),
			"scope":       prometheus.KeyLabel("request.scope"),
			"subresource": prometheus.KeyLabel("request.subresource"),
			"verb":        prometheus.KeyLabel("request.verb"),
		},
	}

	mb.Registry.MustAddMetricSet("kubernetes", "apiserver",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
