// +build darwin linux openbsd windows

package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "cpu", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system CPU metrics.
type MetricSet struct {
	mb.BaseMetricSet
	cpu *system.CPU
}

// New is a mb.MetricSetFactory that returns a cpu.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	return &MetricSet{
		BaseMetricSet: base,
		cpu:           &system.CPU{},
	}, nil
}

// Fetch fetches CPU metrics from the OS.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	cpuStat, err := system.GetCpuTimes()
	if err != nil {
		return nil, errors.Wrap(err, "cpu times")
	}
	m.cpu.AddCpuPercentage(cpuStat)

	loadStat, err := system.GetSystemLoad()
	if err != nil {
		return nil, errors.Wrap(err, "load statistics")
	}

	event := system.GetCpuStatEvent(cpuStat)
	event["load"] = loadStat

	return event, nil
}
