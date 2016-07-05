// +build darwin freebsd linux openbsd windows

package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	system "github.com/elastic/beats/metricbeat/module/system/common"

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

	config := struct {
		CpuTicks bool `config:"cpu_ticks"` // export CPU usage in ticks
	}{
		CpuTicks: false,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		cpu: &system.CPU{
			CpuTicks: config.CpuTicks,
		},
	}, nil
}

// Fetch fetches CPU metrics from the OS.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	stat, err := system.GetCpuTimes()
	if err != nil {
		return nil, errors.Wrap(err, "cpu times")
	}
	m.cpu.AddCpuPercentage(stat)

	loadStat, err := system.GetSystemLoad()
	if err != nil {
		return nil, errors.Wrap(err, "load statistics")
	}

	cpuStat := common.MapStr{
		"user": common.MapStr{
			"pct": stat.UserPercent,
		},
		"system": common.MapStr{
			"pct": stat.SystemPercent,
		},
		"idle": common.MapStr{
			"pct": stat.IdlePercent,
		},
		"iowait": common.MapStr{
			"pct": stat.IOwaitPercent,
		},
		"irq": common.MapStr{
			"pct": stat.IrqPercent,
		},
		"nice": common.MapStr{
			"pct": stat.NicePercent,
		},
		"softirq": common.MapStr{
			"pct": stat.SoftIrqPercent,
		},
		"steal": common.MapStr{
			"pct": stat.StealPercent,
		},
	}

	if m.cpu.CpuTicks {
		cpuStat["user"].(common.MapStr)["ticks"] = stat.User
		cpuStat["system"].(common.MapStr)["ticks"] = stat.Sys
		cpuStat["nice"].(common.MapStr)["ticks"] = stat.Nice
		cpuStat["idle"].(common.MapStr)["ticks"] = stat.Idle
		cpuStat["iowait"].(common.MapStr)["ticks"] = stat.Wait
		cpuStat["irq"].(common.MapStr)["ticks"] = stat.Irq
		cpuStat["softirq"].(common.MapStr)["ticks"] = stat.SoftIrq
		cpuStat["steal"].(common.MapStr)["ticks"] = stat.Stolen
	}

	cpuStat["load"] = common.MapStr{
		"1":  loadStat.Load1,
		"5":  loadStat.Load5,
		"15": loadStat.Load15,
	}

	return cpuStat, nil
}
