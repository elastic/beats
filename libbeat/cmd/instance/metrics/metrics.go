// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build (darwin && cgo) || (freebsd && cgo) || linux || windows
// +build darwin,cgo freebsd,cgo linux windows

package metrics

import (
	"os"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup"
	"github.com/elastic/beats/v7/libbeat/metric/system/cpu"
	"github.com/elastic/beats/v7/libbeat/metric/system/numcpu"
	"github.com/elastic/beats/v7/libbeat/metric/system/process"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

var (
	beatProcessStats *process.Stats
	systemMetrics    *monitoring.Registry
)

// libbeatMonitoringCgroupsHierarchyOverride is an undocumented environment variable which
// overrides the cgroups path under /sys/fs/cgroup, which should be set to "/" when running
// Beats under Docker.
const libbeatMonitoringCgroupsHierarchyOverride = "LIBBEAT_MONITORING_CGROUPS_HIERARCHY_OVERRIDE"

func init() {
	systemMetrics = monitoring.Default.NewRegistry("system")
}

func SetupMetrics(name string) error {
	monitoring.NewFunc(systemMetrics, "cpu", reportSystemCPUUsage, monitoring.Report)

	//if the beat name is longer than 15 characters, truncate it so we don't fail process checks later on
	// On *nix, the process name comes from /proc/PID/stat, which uses a comm value of 16 bytes, plus the null byte
	if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && len(name) > 15 {
		name = name[:15]
	}

	beatProcessStats = &process.Stats{
		Procs:        []string{name},
		EnvWhitelist: nil,
		CPUTicks:     true,
		CacheCmdLine: true,
		IncludeTop:   process.IncludeTopConfig{},
	}

	err := beatProcessStats.Init()
	if err != nil {
		return err
	}

	monitoring.NewFunc(beatMetrics, "memstats", reportMemStats, monitoring.Report)
	monitoring.NewFunc(beatMetrics, "cpu", reportBeatCPU, monitoring.Report)
	monitoring.NewFunc(beatMetrics, "runtime", reportRuntime, monitoring.Report)

	setupPlatformSpecificMetrics()

	return nil
}

func setupPlatformSpecificMetrics() {
	switch runtime.GOOS {
	case "linux":
		monitoring.NewFunc(beatMetrics, "cgroup", reportBeatCgroups, monitoring.Report)
	case "windows":
		setupWindowsHandlesMetrics()
	}

	if runtime.GOOS != "windows" {
		monitoring.NewFunc(systemMetrics, "load", reportSystemLoadAverage, monitoring.Report)
	}

	setupLinuxBSDFDMetrics()
}

func reportMemStats(m monitoring.Mode, V monitoring.Visitor) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	monitoring.ReportInt(V, "memory_total", int64(stats.TotalAlloc))
	if m == monitoring.Full {
		monitoring.ReportInt(V, "memory_alloc", int64(stats.Alloc))
		monitoring.ReportInt(V, "memory_sys", int64(stats.Sys))
		monitoring.ReportInt(V, "gc_next", int64(stats.NextGC))
	}

	rss, err := getRSSSize()
	if err != nil {
		logp.Err("Error while getting memory usage: %v", err)
		return
	}
	monitoring.ReportInt(V, "rss", int64(rss))
}

func getRSSSize() (uint64, error) {
	state, err := beatProcessStats.GetSelf()
	if err != nil {
		return 0, err
	}

	return state.Memory.Rss.Bytes.ValueOr(0), nil
}

func reportBeatCPU(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	state, err := beatProcessStats.GetSelf()
	if err != nil {
		logp.Err("Error retrieving CPU percentages: %v", err)
		return
	}

	monitoring.ReportNamespace(V, "user", func() {
		monitoring.ReportInt(V, "ticks", int64(state.CPU.User.Ticks.ValueOr(0)))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", int64(state.CPU.User.Ticks.ValueOr(0)))
		})
	})
	monitoring.ReportNamespace(V, "system", func() {
		monitoring.ReportInt(V, "ticks", int64(state.CPU.System.Ticks.ValueOr(0)))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", int64(state.CPU.System.Ticks.ValueOr(0)))
		})
	})
	monitoring.ReportNamespace(V, "total", func() {
		monitoring.ReportFloat(V, "value", state.CPU.Total.Value.ValueOr(0))
		monitoring.ReportInt(V, "ticks", int64(state.CPU.Total.Ticks.ValueOr(0)))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", int64(state.CPU.Total.Ticks.ValueOr(0)))
		})
	})
}

func reportSystemLoadAverage(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	load, err := cpu.Load()
	if err != nil {
		logp.Err("Error retrieving load average: %v", err)
		return
	}
	avgs := load.Averages()
	monitoring.ReportFloat(V, "1", avgs.OneMinute)
	monitoring.ReportFloat(V, "5", avgs.FiveMinute)
	monitoring.ReportFloat(V, "15", avgs.FifteenMinute)

	normAvgs := load.NormalizedAverages()
	monitoring.ReportNamespace(V, "norm", func() {
		monitoring.ReportFloat(V, "1", normAvgs.OneMinute)
		monitoring.ReportFloat(V, "5", normAvgs.FiveMinute)
		monitoring.ReportFloat(V, "15", normAvgs.FifteenMinute)
	})
}

func reportSystemCPUUsage(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	monitoring.ReportInt(V, "cores", int64(numcpu.NumCPU()))
}

func reportRuntime(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	monitoring.ReportInt(V, "goroutines", int64(runtime.NumGoroutine()))
}

func reportBeatCgroups(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	pid := os.Getpid()

	cgroups, err := cgroup.NewReaderOptions(cgroup.ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("/"),
		IgnoreRootCgroups:        true,
		CgroupsHierarchyOverride: os.Getenv(libbeatMonitoringCgroupsHierarchyOverride),
	})
	if err != nil {
		if err == cgroup.ErrCgroupsMissing {
			logp.Warn("cgroup data collection disabled: %v", err)
		} else {
			logp.Err("cgroup data collection disabled: %v", err)
		}
		return
	}

	cgv, err := cgroups.CgroupsVersion(pid)
	if err != nil {
		logp.Err("error determining cgroups version: %v", err)
		return
	}

	if cgv == cgroup.CgroupsV1 {
		reportMetricsCGV1(pid, cgroups, V)
	} else {
		reportMetricsCGV2(pid, cgroups, V)
	}

}

func reportMetricsCGV1(pid int, cgroups *cgroup.Reader, V monitoring.Visitor) {
	selfStats, err := cgroups.GetV1StatsForProcess(pid)
	if err != nil {
		logp.Err("error getting cgroup stats: %v", err)
	}
	// GetStatsForProcess returns a nil selfStats and no error when there's no stats
	if selfStats == nil {
		return
	}

	if cpu := selfStats.CPU; cpu != nil {
		monitoring.ReportNamespace(V, "cpu", func() {
			if cpu.ID != "" {
				monitoring.ReportString(V, "id", cpu.ID)
			}
			monitoring.ReportNamespace(V, "cfs", func() {
				monitoring.ReportNamespace(V, "period", func() {
					monitoring.ReportInt(V, "us", int64(cpu.CFS.PeriodMicros.Us))
				})
				monitoring.ReportNamespace(V, "quota", func() {
					monitoring.ReportInt(V, "us", int64(cpu.CFS.QuotaMicros.Us))
				})
			})
			monitoring.ReportNamespace(V, "stats", func() {
				monitoring.ReportInt(V, "periods", int64(cpu.Stats.Periods))
				monitoring.ReportNamespace(V, "throttled", func() {
					monitoring.ReportInt(V, "periods", int64(cpu.Stats.Throttled.Periods))
					monitoring.ReportInt(V, "ns", int64(cpu.Stats.Throttled.Us))
				})
			})
		})
	}

	if cpuacct := selfStats.CPUAccounting; cpuacct != nil {
		monitoring.ReportNamespace(V, "cpuacct", func() {
			if cpuacct.ID != "" {
				monitoring.ReportString(V, "id", cpuacct.ID)
			}
			monitoring.ReportNamespace(V, "total", func() {
				monitoring.ReportInt(V, "ns", int64(cpuacct.Total.NS))
			})
		})
	}

	if memory := selfStats.Memory; memory != nil {
		monitoring.ReportNamespace(V, "memory", func() {
			if memory.ID != "" {
				monitoring.ReportString(V, "id", memory.ID)
			}
			monitoring.ReportNamespace(V, "mem", func() {
				monitoring.ReportNamespace(V, "limit", func() {
					monitoring.ReportInt(V, "bytes", int64(memory.Mem.Limit.Bytes))
				})
				monitoring.ReportNamespace(V, "usage", func() {
					monitoring.ReportInt(V, "bytes", int64(memory.Mem.Usage.Bytes))
				})
			})
		})
	}
}

func reportMetricsCGV2(pid int, cgroups *cgroup.Reader, V monitoring.Visitor) {
	selfStats, err := cgroups.GetV2StatsForProcess(pid)
	if err != nil {
		logp.Err("error getting cgroup stats: %v", err)
		return
	}
	if cpu := selfStats.CPU; cpu != nil {
		monitoring.ReportNamespace(V, "cpu", func() {
			if cpu.ID != "" {
				monitoring.ReportString(V, "id", cpu.ID)
			}
			monitoring.ReportNamespace(V, "stats", func() {
				monitoring.ReportInt(V, "periods", int64(cpu.Stats.Periods.ValueOr(0)))
				monitoring.ReportNamespace(V, "throttled", func() {
					monitoring.ReportInt(V, "periods", int64(cpu.Stats.Throttled.Periods.ValueOr(0)))
					monitoring.ReportInt(V, "ns", int64(cpu.Stats.Throttled.Us.ValueOr(0)))
				})
			})
		})
	}

	if memory := selfStats.Memory; memory != nil {
		monitoring.ReportNamespace(V, "memory", func() {
			if memory.ID != "" {
				monitoring.ReportString(V, "id", memory.ID)
			}
			monitoring.ReportNamespace(V, "mem", func() {
				monitoring.ReportNamespace(V, "usage", func() {
					monitoring.ReportInt(V, "bytes", int64(memory.Mem.Usage.Bytes))
				})
			})
		})
	}

}
