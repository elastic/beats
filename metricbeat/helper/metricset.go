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
func NewMetricSet(name string, new func() MetricSeter, module *Module) (*MetricSet, error) {
	metricSeter := new()

	ms := &MetricSet{
		Name:        name,
		MetricSeter: metricSeter,
		Config:      module.Config,
		Module:      module,
	}

	return ms, nil
}

func (m *MetricSet) Setup() error {
	// TODO: Call fetch for each host if hosts are set.
	// Host is a first class citizen and does not have to be handled by the metricset itself
	return m.MetricSeter.Setup(m)
}

// RunMetric runs the given metricSet and returns the event
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// TODO: Call fetch for each host if hosts are set.
	// Host is a first class citizen and does not have to be handled by the metricset itself
	return m.MetricSeter.Fetch(m)
}
