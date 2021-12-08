package memory

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
			"container_memory_usage_max_bytes":           prometheus.Metric("usage.max"),
			"container_memory_usage_usage_bytes":         prometheus.Metric("usage.total"),
			"container_memory_total_inactive_file_bytes": prometheus.Metric("usage.inactiveFiles"),
			"container_memory_usage_limit_bytes":         prometheus.Metric("limit"),
		},
		Labels: map[string]prometheus.LabelMap{
			"container_id": prometheus.KeyLabel("id"),
		},
	}

	mb.Registry.MustAddMetricSet("containerd", "memory",
		getMetricsetFactory(mapping),
		mb.WithHostParser(hostParser),
	)
}
