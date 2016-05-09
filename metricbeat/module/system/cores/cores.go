// +build darwin linux openbsd windows

package cores

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "cores", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system core metrics.
type MetricSet struct {
	mb.BaseMetricSet
	cpu *system.CPU
}

// New is a mb.MetricSetFactory that returns a cores.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	return &MetricSet{
		BaseMetricSet: base,
		cpu: &system.CPU{
			CpuPerCore: true,
		},
	}, nil
}

// Fetch fetches CPU core metrics from the OS.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	cpuCoreStat, err := system.GetCpuTimesList()
	if err != nil {
		return nil, errors.Wrap(err, "cpu core times")
	}
	m.cpu.AddCpuPercentageList(cpuCoreStat)

	cores := []common.MapStr{}

	for core, stat := range cpuCoreStat {
		coreStat := system.GetCpuStatEvent(&stat)
		coreStat["core"] = core
		cores = append(cores, coreStat)

	}

	return cores, nil
}
