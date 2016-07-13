// +build darwin freebsd linux windows

package process

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "process", New); err != nil {
		panic(err)
	}
}

// MetricSet that fetches process metrics.
type MetricSet struct {
	mb.BaseMetricSet
	stats *ProcStats
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Procs []string `config:"processes"` // collect all processes by default
	}{
		Procs: []string{".*"},
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet: base,
		stats: &ProcStats{
			ProcStats: true,
			Procs:     config.Procs,
		},
	}
	err := m.stats.InitProcStats()
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Fetch fetches metrics for all processes. It iterates over each PID and
// collects process metadata, CPU metrics, and memory metrics.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	procs, err := m.stats.GetProcStats()
	if err != nil {
		return nil, errors.Wrap(err, "process stats")
	}

	return procs, err
}
