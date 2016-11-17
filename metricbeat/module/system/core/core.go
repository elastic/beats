// +build darwin freebsd linux openbsd windows

package core

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system/cpu"

	"github.com/pkg/errors"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "core", New); err != nil {
		panic(err)
	}
}

// MetricSet for fetching system core metrics.
type MetricSet struct {
	mb.BaseMetricSet
	cpu *cpu.CPU
}

// New is a mb.MetricSetFactory that returns a cores.MetricSet.
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
		cpu: &cpu.CPU{
			CpuPerCore: true,
			CpuTicks:   config.CpuTicks,
		},
	}, nil
}

// Fetch fetches CPU core metrics from the OS.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	cpuCoreStat, err := cpu.GetCpuTimesList()
	if err != nil {
		return nil, errors.Wrap(err, "cpu core times")
	}

	m.cpu.AddCpuPercentageList(cpuCoreStat)

	cores := make([]common.MapStr, 0, len(cpuCoreStat))
	for core, stat := range cpuCoreStat {

		coreStat := common.MapStr{
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
			coreStat["user"].(common.MapStr)["ticks"] = stat.User
			coreStat["system"].(common.MapStr)["ticks"] = stat.Sys
			coreStat["nice"].(common.MapStr)["ticks"] = stat.Nice
			coreStat["idle"].(common.MapStr)["ticks"] = stat.Idle
			coreStat["iowait"].(common.MapStr)["ticks"] = stat.Wait
			coreStat["irq"].(common.MapStr)["ticks"] = stat.Irq
			coreStat["softirq"].(common.MapStr)["ticks"] = stat.SoftIrq
			coreStat["steal"].(common.MapStr)["ticks"] = stat.Stolen
		}

		coreStat["id"] = core
		cores = append(cores, coreStat)
	}

	return cores, nil
}
