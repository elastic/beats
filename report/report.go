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

package report

import (
	"context"
	"errors"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v4/common"
	psprocess "github.com/shirou/gopsutil/v4/process"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cpu"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/numcpu"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/process"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func MemStatsReporter(logger *logp.Logger, processStats *process.Stats) func(monitoring.Mode, monitoring.Visitor) {
	pid, err := process.GetSelfPid(processStats.Hostfs)
	if err != nil {
		logger.Errorf("Error while retrieving pid: %v", err)
		return nil
	}
	p := psprocess.Process{
		Pid: int32(pid),
	}

	ctx := context.Background()
	if processStats != nil && processStats.Hostfs != nil && processStats.Hostfs.IsSet() {
		ctx = context.WithValue(context.Background(), common.EnvKey, common.EnvMap{common.HostProcEnvKey: processStats.Hostfs.ResolveHostFS("")})
	}

	return func(m monitoring.Mode, V monitoring.Visitor) {
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

		statm, err := p.MemoryInfoWithContext(ctx)
		if err != nil {
			logger.Errorf("Error while getting memory usage: %v", err)
			return
		}

		monitoring.ReportInt(V, "rss", int64(statm.RSS))
	}
}

func InstanceCPUReporter(logger *logp.Logger, processStats *process.Stats) func(monitoring.Mode, monitoring.Visitor) {
	pid, err := process.GetSelfPid(processStats.Hostfs)
	if err != nil {
		logger.Errorf("Error while retrieving pid: %v", err)
		return nil
	}
	p := psprocess.Process{
		Pid: int32(pid),
	}

	ctx := context.Background()
	if processStats != nil && processStats.Hostfs != nil && processStats.Hostfs.IsSet() {
		ctx = context.WithValue(context.Background(), common.EnvKey, common.EnvMap{common.HostProcEnvKey: processStats.Hostfs.ResolveHostFS("")})
	}

	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		stat, err := p.TimesWithContext(ctx)
		if err != nil {
			logger.Errorf("Error retrieving CPU percentages: %v", err)
			return
		}

		userMs := stat.User * 1000
		sysMs := stat.System * 1000
		totalMs := userMs + sysMs

		monitoring.ReportNamespace(V, "user", func() {
			monitoring.ReportInt(V, "ticks", int64(userMs))
			monitoring.ReportNamespace(V, "time", func() {
				monitoring.ReportInt(V, "ms", int64(userMs))
			})
		})
		monitoring.ReportNamespace(V, "system", func() {
			monitoring.ReportInt(V, "ticks", int64(sysMs))
			monitoring.ReportNamespace(V, "time", func() {
				monitoring.ReportInt(V, "ms", int64(sysMs))
			})
		})
		monitoring.ReportNamespace(V, "total", func() {
			monitoring.ReportFloat(V, "value", totalMs)
			monitoring.ReportInt(V, "ticks", int64(totalMs))
			monitoring.ReportNamespace(V, "time", func() {
				monitoring.ReportInt(V, "ms", int64(totalMs))
			})
		})
	}
}

func ReportSystemLoadAverage(logger *logp.Logger) func(monitoring.Mode, monitoring.Visitor) {
	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		load, err := cpu.LoadWithLogger(logger)
		if err != nil {
			logger.Errorf("Error retrieving load average: %v", err)
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
}

func ReportSystemCPUUsage(logger *logp.Logger) func(monitoring.Mode, monitoring.Visitor) {
	return func(m monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		monitoring.ReportInt(V, "cores", int64(numcpu.NumCPUWithLogger(logger)))
	}
}

func ReportRuntime(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	monitoring.ReportInt(V, "goroutines", int64(runtime.NumGoroutine()))
}

func InstanceCroupsReporter(logger *logp.Logger, override string) func(monitoring.Mode, monitoring.Visitor) {
	// PID shouldn't use hostfs, at least for now.
	// containerization schemes should provide their own /proc/ that will serve containerized processess
	pid := os.Getpid()

	cgroups, err := cgroup.NewReaderOptions(cgroup.ReaderOptions{
		RootfsMountpoint:         resolve.NewTestResolver("/"),
		IgnoreRootCgroups:        true,
		CgroupsHierarchyOverride: os.Getenv(override),
		Logger:                   logger,
	})
	if err != nil {
		if errors.Is(err, cgroup.ErrCgroupsMissing) {
			logger.Warnf("cgroup data collection disabled in internal monitoring: %v", err)
		} else {
			logger.Errorf("cgroup data collection disabled in internal monitoring: %v", err)
		}
		return func(m monitoring.Mode, v monitoring.Visitor) {}
	}

	cgv, err := cgroups.CgroupsVersion(pid)
	if err != nil {
		logger.Errorf("error determining cgroups version for internal monitoring: %v", err)
		return func(m monitoring.Mode, v monitoring.Visitor) {}
	}

	return func(_ monitoring.Mode, V monitoring.Visitor) {
		V.OnRegistryStart()
		defer V.OnRegistryFinished()

		if cgv == cgroup.CgroupsV1 {
			ReportMetricsCGV1(logger, pid, cgroups, V)
		} else {
			ReportMetricsCGV2(logger, pid, cgroups, V)
		}
	}
}

func ReportMetricsCGV1(logger *logp.Logger, pid int, cgroups *cgroup.Reader, V monitoring.Visitor) {
	selfStats, err := cgroups.GetV1StatsForProcess(pid)
	if err != nil {
		logger.Errorf("error getting cgroup stats for V1: %v", err)
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

func ReportMetricsCGV2(logger *logp.Logger, pid int, cgroups *cgroup.Reader, V monitoring.Visitor) {
	selfStats, err := cgroups.GetV2StatsForProcess(pid)
	if err != nil {
		logger.Errorf("error getting cgroup stats for V2: %v", err)
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
