package state_container

import (
	"github.com/elastic/beats/libbeat/common"
	p "github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
	// Nanocores conversion 10^9
	nanocores = 1000000000
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()

	mapping = &p.MetricsMapping{
		Metrics: map[string]p.MetricMap{
			"kube_pod_container_info":                           p.Metric(""),
			"kube_pod_container_resource_limits_cpu_cores":      p.Metric("cpu.limit.cores"),
			"kube_pod_container_resource_requests_cpu_cores":    p.Metric("cpu.request.cores"),
			"kube_pod_container_resource_limits_memory_bytes":   p.Metric("memory.limit.bytes"),
			"kube_pod_container_resource_requests_memory_bytes": p.Metric("memory.request.bytes"),
			"kube_pod_container_status_ready":                   p.BooleanMetric("status.ready"),
			"kube_pod_container_status_restarts":                p.Metric("status.restarts"),
			"kube_pod_container_status_running":                 p.KeywordMetric("status.phase", "running"),
			"kube_pod_container_status_terminated":              p.KeywordMetric("status.phase", "terminated"),
			"kube_pod_container_status_waiting":                 p.KeywordMetric("status.phase", "waiting"),
		},

		Labels: map[string]p.LabelMap{
			"pod":       p.KeyLabel(mb.ModuleDataKey + ".pod.name"),
			"container": p.KeyLabel("name"),
			"namespace": p.KeyLabel(mb.ModuleDataKey + ".namespace"),

			"node":         p.Label(mb.ModuleDataKey + ".node.name"),
			"container_id": p.Label("id"),
			"image":        p.Label("image"),
		},

		ExtraFields: map[string]string{
			mb.NamespaceKey: "container",
		},
	}
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kubernetes", "state_container", New, hostParser); err != nil {
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
	events, err := m.prometheus.GetProcessedMetrics(mapping)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		if request, ok := event["cpu.request.cores"]; ok {
			if requestCores, ok := request.(float64); ok {
				event["cpu.request.nanocores"] = requestCores * nanocores
			}
		}

		if limit, ok := event["cpu.limit.cores"]; ok {
			if limitCores, ok := limit.(float64); ok {
				event["cpu.limit.nanocores"] = limitCores * nanocores
			}
		}
	}

	return events, err
}
