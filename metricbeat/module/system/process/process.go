// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"runtime"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system"

	"github.com/elastic/gosigar/cgroup"
	"github.com/pkg/errors"
)

var debugf = logp.MakeDebug("system-process")

func init() {
	if err := mb.Registry.AddMetricSet("system", "process", New); err != nil {
		panic(err)
	}
}

// MetricSet that fetches process metrics.
type MetricSet struct {
	mb.BaseMetricSet
	stats  *ProcStats
	cgroup *cgroup.Reader
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Procs   []string `config:"processes"` // collect all processes by default
		Cgroups bool     `config:"cgroups"`
	}{
		Procs:   []string{".*"},
		Cgroups: false,
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

	if runtime.GOOS == "linux" {
		systemModule, ok := base.Module().(*system.Module)
		if !ok {
			return nil, fmt.Errorf("unexpected module type")
		}

		if config.Cgroups {
			logp.Warn("EXPERIMENTAL: Cgroup is enabled for the system.process MetricSet.")
			m.cgroup, err = cgroup.NewReader(systemModule.HostFS, true)
			if err != nil {
				return nil, errors.Wrap(err, "error initializing cgroup reader")
			}
		}
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

	if m.cgroup != nil {
		for _, proc := range procs {
			pid, ok := proc["pid"].(int)
			if !ok {
				debugf("error converting pid to int for proc %+v", proc)
				continue
			}
			stats, err := m.cgroup.GetStatsForProcess(pid)
			if err != nil {
				debugf("error getting cgroups stats for pid=%d, %v", pid, err)
				continue
			}

			if statsMap := cgroupStatsToMap(stats); statsMap != nil {
				proc["cgroup"] = statsMap
			}
		}
	}

	return procs, err
}
