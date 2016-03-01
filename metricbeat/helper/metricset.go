package helper

import (
	"github.com/elastic/beats/libbeat/common"
)

// Metric specific data
// This must be defined by each metric
type MetricSet struct {
	Name        string
	MetricSeter MetricSeter
	// Inherits config from module
	Config ModuleConfig
	Module *Module
}

// Creates a new MetricSet
func NewMetricSet(name string, metricset MetricSeter, module *Module) *MetricSet {
	return &MetricSet{
		Name:        name,
		MetricSeter: metricset,
		Config:      module.Config,
		Module:      module,
	}
}

// RunMetric runs the given metricSet and returns the event
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// TODO: Call fetch for each host if hosts are set.
	// Host is a first class citizen and does not have to be handled by the metricset itself
	return m.MetricSeter.Fetch(m)
}
