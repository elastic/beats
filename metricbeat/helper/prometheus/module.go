package prometheus

import (
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	// HostParser validates Prometheus URLs
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSetBuilder returns a builder function for a new Prometheus metricset using the given mapping
func MetricSetBuilder(mapping *MetricsMapping) func(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		prometheus, err := NewPrometheusClient(base)
		if err != nil {
			return nil, err
		}
		return &prometheusMetricSet{
			BaseMetricSet: base,
			prometheus:    prometheus,
			mapping:       mapping,
		}, nil
	}
}

type prometheusMetricSet struct {
	mb.BaseMetricSet
	prometheus Prometheus
	mapping    *MetricsMapping
}

func (m *prometheusMetricSet) Fetch(r mb.ReporterV2) {
	m.prometheus.ReportProcessedMetrics(m.mapping, r)
}
