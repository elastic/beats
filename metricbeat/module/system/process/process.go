// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/system"
	"github.com/elastic/gosigar/cgroup"
)

var debugf = logp.MakeDebug("system.process")

func init() {
	mb.Registry.MustAddMetricSet("system", "process", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet that fetches process metrics.
type MetricSet struct {
	mb.BaseMetricSet
	stats        *process.Stats
	cgroup       *cgroup.Reader
	cacheCmdLine bool
}

// New creates and returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet: base,
		stats: &process.Stats{
			Procs:        config.Procs,
			EnvWhitelist: config.EnvWhitelist,
			CpuTicks:     config.IncludeCPUTicks || (config.CPUTicks != nil && *config.CPUTicks),
			CacheCmdLine: config.CacheCmdLine,
			IncludeTop:   config.IncludeTop,
		},
	}
	err := m.stats.Init()
	if err != nil {
		return nil, err
	}

	if runtime.GOOS == "linux" {
		systemModule, ok := base.Module().(*system.Module)
		if !ok {
			return nil, fmt.Errorf("unexpected module type")
		}

		if config.Cgroups == nil || *config.Cgroups {
			debugf("process cgroup data collection is enabled, using hostfs='%v'", systemModule.HostFS)
			m.cgroup, err = cgroup.NewReader(systemModule.HostFS, true)
			if err != nil {
				if err == cgroup.ErrCgroupsMissing {
					logp.Warn("cgroup data collection will be disabled: %v", err)
				} else {
					return nil, errors.Wrap(err, "error initializing cgroup reader")
				}
			}
		}
	}

	return m, nil
}

// Fetch fetches metrics for all processes. It iterates over each PID and
// collects process metadata, CPU metrics, and memory metrics.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	procs, err := m.stats.Get()
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
