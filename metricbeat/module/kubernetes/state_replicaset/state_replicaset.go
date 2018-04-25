package state_replicaset

import (
	"github.com/elastic/beats/libbeat/common"
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
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

	mapping = &p.MetricsMapping{
		Metrics: map[string]p.MetricMap{
			"kube_replicaset_metadata_generation":           p.Metric(""),
			"kube_replicaset_status_fully_labeled_replicas": p.Metric("replicas.labeled"),
			"kube_replicaset_status_observed_generation":    p.Metric("replicas.observed"),
			"kube_replicaset_status_ready_replicas":         p.Metric("replicas.ready"),
			"kube_replicaset_spec_replicas":                 p.Metric("replicas.desired"),
			"kube_replicaset_status_replicas":               p.Metric("replicas.available"),
		},

		Labels: map[string]p.LabelMap{
			"replicaset": p.KeyLabel("name"),
			"namespace":  p.KeyLabel(mb.ModuleDataKey + ".namespace"),
		},

		ExtraFields: map[string]string{
			mb.NamespaceKey: "replicaset",
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kubernetes", "state_replicaset", New, hostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, err := p.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	return m.prometheus.GetProcessedMetrics(mapping)
}
