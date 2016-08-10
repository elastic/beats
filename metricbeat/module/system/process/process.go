// +build darwin freebsd linux windows

package process

import (
	"fmt"
	"runtime"
	"strconv"

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

	if runtime.GOOS == "linux" {
		systemModule, ok := base.Module().(*system.Module)
		if !ok {
			return nil, fmt.Errorf("unexpected module type")
		}

		m.cgroup, err = cgroup.NewReader(systemModule.HostFS, true)
		if err != nil {
			return nil, errors.Wrap(err, "error initializing cgroup reader")
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

// cgroupStatsToMap returns a MapStr containing the data from the stats object.
// If stats is nil then nil is returned.
func cgroupStatsToMap(stats *cgroup.Stats) common.MapStr {
	if stats == nil {
		return nil
	}

	cgroup := common.MapStr{}

	// id and path are only available when all subsystems share a common path.
	if stats.ID != "" {
		cgroup["id"] = stats.ID
	}
	if stats.Path != "" {
		cgroup["path"] = stats.Path
	}

	if cpu := cgroupCPUToMapStr(stats.CPU); cpu != nil {
		cgroup["cpu"] = cpu
	}
	if cpuacct := cgroupCPUAccountingToMapStr(stats.CPUAccounting); cpuacct != nil {
		cgroup["cpuacct"] = cpuacct
	}
	if memory := cgroupMemoryToMapStr(stats.Memory); memory != nil {
		cgroup["memory"] = memory
	}
	if blkio := cgroupBlockIOToMapStr(stats.BlockIO); blkio != nil {
		cgroup["blkio"] = blkio
	}

	return cgroup
}

// cgroupCPUToMapStr returns a MapStr containing CPUSubsystem data. If the
// cpu parameter is nil then nil is returned.
func cgroupCPUToMapStr(cpu *cgroup.CPUSubsystem) common.MapStr {
	if cpu == nil {
		return nil
	}

	return common.MapStr{
		"id":   cpu.ID,
		"path": cpu.Path,
		"cfs": common.MapStr{
			"period": common.MapStr{
				"us": cpu.CFS.PeriodMicros,
			},
			"quota": common.MapStr{
				"us": cpu.CFS.QuotaMicros,
			},
			"shares": cpu.CFS.Shares,
		},
		"rt": common.MapStr{
			"period": common.MapStr{
				"us": cpu.RT.PeriodMicros,
			},
			"runtime": common.MapStr{
				"us": cpu.RT.RuntimeMicros,
			},
		},
		"stats": common.MapStr{
			"periods": cpu.Stats.Periods,
			"throttled": common.MapStr{
				"periods": cpu.Stats.ThrottledPeriods,
				"nanos":   cpu.Stats.ThrottledTimeNanos,
			},
		},
	}
}

// cgroupCPUAccountingToMapStr returns a MapStr containing
// CPUAccountingSubsystem data. If the cpuacct parameter is nil then nil is
// returned.
func cgroupCPUAccountingToMapStr(cpuacct *cgroup.CPUAccountingSubsystem) common.MapStr {
	if cpuacct == nil {
		return nil
	}

	perCPUUsage := common.MapStr{}
	for i, usage := range cpuacct.UsagePerCPU {
		perCPUUsage[strconv.Itoa(i+1)] = usage
	}

	return common.MapStr{
		"id":   cpuacct.ID,
		"path": cpuacct.Path,
		"total": common.MapStr{
			"nanos": cpuacct.TotalNanos,
		},
		"percpu": perCPUUsage,
		"stats": common.MapStr{
			"system": common.MapStr{
				"nanos": cpuacct.Stats.SystemNanos,
			},
			"user": common.MapStr{
				"nanos": cpuacct.Stats.UserNanos,
			},
		},
	}
}

// cgroupMemoryToMapStr returns a MapStr containing MemorySubsystem data. If the
// memory parameter is nil then nil is returned.
func cgroupMemoryToMapStr(memory *cgroup.MemorySubsystem) common.MapStr {
	if memory == nil {
		return nil
	}

	addMemData := func(key string, m common.MapStr, data cgroup.MemoryData) {
		m[key] = common.MapStr{
			"failures": memory.Mem.FailCount,
			"limit": common.MapStr{
				"bytes": memory.Mem.Limit,
			},
			"usage": common.MapStr{
				"bytes": memory.Mem.Usage,
				"max": common.MapStr{
					"bytes": memory.Mem.MaxUsage,
				},
			},
		}
	}

	memMap := common.MapStr{
		"id":   memory.ID,
		"path": memory.Path,
	}
	addMemData("mem", memMap, memory.Mem)
	addMemData("memsw", memMap, memory.MemSwap)
	addMemData("kmem", memMap, memory.Kernel)
	addMemData("kmem_tcp", memMap, memory.KernelTCP)
	memMap["stats"] = common.MapStr{
		"active_anon": common.MapStr{
			"bytes": memory.Stats.ActiveAnon,
		},
		"active_file": common.MapStr{
			"bytes": memory.Stats.ActiveFile,
		},
		"cache": common.MapStr{
			"bytes": memory.Stats.Cache,
		},
		"hierarchical_memory_limit": common.MapStr{
			"bytes": memory.Stats.HierarchicalMemoryLimit,
		},
		"hierarchical_memsw_limit": common.MapStr{
			"bytes": memory.Stats.HierarchicalMemswLimit,
		},
		"inactive_anon": common.MapStr{
			"bytes": memory.Stats.InactiveAnon,
		},
		"inactive_file": common.MapStr{
			"bytes": memory.Stats.InactiveFile,
		},
		"mapped_file": common.MapStr{
			"bytes": memory.Stats.MappedFile,
		},
		"page_faults":       memory.Stats.PageFaults,
		"major_page_faults": memory.Stats.MajorPageFaults,
		"pages_in":          memory.Stats.PagesIn,
		"pages_out":         memory.Stats.PagesOut,
		"rss": common.MapStr{
			"bytes": memory.Stats.RSS,
		},
		"rss_huge": common.MapStr{
			"bytes": memory.Stats.RSSHuge,
		},
		"swap": common.MapStr{
			"bytes": memory.Stats.Swap,
		},
		"unevictable": common.MapStr{
			"bytes": memory.Stats.Unevictable,
		},
	}

	return memMap
}

// cgroupBlockIOToMapStr returns a MapStr containing BlockIOSubsystem data.
// If the blockIO parameter is nil then nil is returned.
func cgroupBlockIOToMapStr(blockIO *cgroup.BlockIOSubsystem) common.MapStr {
	if blockIO == nil {
		return nil
	}

	return common.MapStr{
		"id":   blockIO.ID,
		"path": blockIO.Path,
		"total": common.MapStr{
			"bytes": blockIO.Throttle.TotalBytes,
			"ios":   blockIO.Throttle.TotalIOs,
		},
	}
}
