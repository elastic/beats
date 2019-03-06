package metrics

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

func init() {
	mapping := &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"etcd_server_has_leader": prometheus.Metric("has_leader"),
		},
	}

	mb.Registry.MustAddMetricSet("etcd", "metrics",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(prometheus.HostParser))
}
