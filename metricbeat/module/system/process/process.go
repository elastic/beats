// +build darwin linux windows

package process

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"

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
	stats *system.ProcStats
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	m := &MetricSet{
		BaseMetricSet: base,
		stats: &system.ProcStats{
			ProcStats: true,
			Procs:     []string{".*"}, // Collect all processes.
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
