// +build windows

package perfmon

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

type CounterConfig struct {
	Name  string               `config:"group" validate:"required"`
	Group []CounterConfigGroup `config:"collectors" validate:"required"`
}

type CounterConfigGroup struct {
	Alias string `config:"alias" validate:"required"`
	Query string `config:"query" validate:"required"`
}

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("windows", "perfmon", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	counters   []CounterConfig
	handle     *Handle
	firstFetch bool
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("BETA: The perfmon metricset is beta")

	config := struct {
		CounterConfig []CounterConfig `config:"perfmon.counters" validate:"required"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	query, err := GetHandle(config.CounterConfig)

	if err != ERROR_SUCCESS {
		return nil, errors.New("initialization fails with error: " + err.Error())
	}

	return &MetricSet{
		BaseMetricSet: base,
		counters:      config.CounterConfig,
		handle:        query,
		firstFetch:    true,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	data, err := m.handle.ReadData(m.firstFetch)
	if err != ERROR_SUCCESS {
		return nil, errors.New("fetching fails wir error: " + err.Error())
	}

	if m.firstFetch {
		m.firstFetch = false
	}

	return data, nil
}
