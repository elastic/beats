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
	"fmt"
	"os"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common"
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
	state, err := getBeatProcessState()
	if err != nil {
		return 0, err
	}

	iRss, err := state.GetValue("memory.rss.bytes")
	if err != nil {
		return 0, fmt.Errorf("error getting Resident Set Size: %v", err)
	}

	rss, ok := iRss.(uint64)
	if !ok {
		return 0, fmt.Errorf("error converting Resident Set Size to uint64: %v", iRss)
	}
	return rss, nil
}

func getBeatProcessState() (common.MapStr, error) {
	pid, err := process.GetSelfPid()
	if err != nil {
		return nil, fmt.Errorf("error getting PID for self process: %v", err)
	}

	state, err := beatProcessStats.GetOne(pid)
	if err != nil {
		return nil, fmt.Errorf("error retrieving process stats: %v", err)
	}

	return state, nil
}

func reportBeatCPU(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	totalCPUUsage, cpuTicks, err := getCPUUsage()
	if err != nil {
		logp.Err("Error retrieving CPU percentages: %v", err)
		return
	}

	userTime, systemTime, err := process.GetOwnResourceUsageTimeInMillis()
	if err != nil {
		logp.Err("Error retrieving CPU usage time: %v", err)
		return
	}

	monitoring.ReportNamespace(V, "user", func() {
		monitoring.ReportInt(V, "ticks", int64(cpuTicks.User))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", userTime)
		})
	})
	monitoring.ReportNamespace(V, "system", func() {
		monitoring.ReportInt(V, "ticks", int64(cpuTicks.System))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", systemTime)
		})
	})
	monitoring.ReportNamespace(V, "total", func() {
		monitoring.ReportFloat(V, "value", totalCPUUsage)
		monitoring.ReportInt(V, "ticks", int64(cpuTicks.Total))
		monitoring.ReportNamespace(V, "time", func() {
			monitoring.ReportInt(V, "ms", userTime+systemTime)
		})
	})
}

func getCPUUsage() (float64, *process.Ticks, error) {
	state, err := getBeatProcessState()
	if err != nil {
		return 0.0, nil, err
	}

	iTotalCPUUsage, err := state.GetValue("cpu.total.value")
	if err != nil {
		return 0.0, nil, fmt.Errorf("error getting total CPU since start: %v", err)
	}

	totalCPUUsage, ok := iTotalCPUUsage.(float64)
	if !ok {
		return 0.0, nil, fmt.Errorf("error converting value of CPU usage since start to float64: %v", iTotalCPUUsage)
	}

	iTotalCPUUserTicks, err := state.GetValue("cpu.user.ticks")
	if err != nil {
		return 0.0, nil, fmt.Errorf("error getting number of user CPU ticks since start: %v", err)
	}

	totalCPUUserTicks, ok := iTotalCPUUserTicks.(uint64)
	if !ok {
		return 0.0, nil, fmt.Errorf("error converting value of user CPU ticks since start to uint64: %v", iTotalCPUUserTicks)
	}

	iTotalCPUSystemTicks, err := state.GetValue("cpu.system.ticks")
	if err != nil {
		return 0.0, nil, fmt.Errorf("error getting number of system CPU ticks since start: %v", err)
	}

	totalCPUSystemTicks, ok := iTotalCPUSystemTicks.(uint64)
	if !ok {
		return 0.0, nil, fmt.Errorf("error converting value of system CPU ticks since start to uint64: %v", iTotalCPUSystemTicks)
	}

	iTotalCPUTicks, err := state.GetValue("cpu.total.ticks")
	if err != nil {
		return 0.0, nil, fmt.Errorf("error getting total number of CPU ticks since start: %v", err)
	}

	totalCPUTicks, ok := iTotalCPUTicks.(uint64)
	if !ok {
		return 0.0, nil, fmt.Errorf("error converting total value of CPU ticks since start to uint64: %v", iTotalCPUTicks)
	}

	p := process.Ticks{
		User:   totalCPUUserTicks,
		System: totalCPUSystemTicks,
		Total:  totalCPUTicks,
	}

	return totalCPUUsage, &p, nil
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

	// PID shouldn't use hostfs, at least for now.
	// containerization schemes should provide their own /proc/ that will serve containerized processess
	pid := os.Getpid()

	cgroups, err := cgroup.NewReaderOptions(cgroup.ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("/"),
		IgnoreRootCgroups:        true,
		CgroupsHierarchyOverride: os.Getenv(libbeatMonitoringCgroupsHierarchyOverride),
	})
	if err != nil {
		if err == cgroup.ErrCgroupsMissing {
			logp.Warn("cgroup data collection disabled in internal monitoring: %v", err)
		} else {
			logp.Err("cgroup data collection disabled in internal monitoring: %v", err)
		}
		return
	}

	cgv, err := cgroups.CgroupsVersion(pid)
	if err != nil {
		logp.Err("error determining cgroups version for internal monitoring: %v", err)
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
		logp.Err("error getting cgroup stats for V1: %v", err)
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
		logp.Err("error getting cgroup stats for V2: %v", err)
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
