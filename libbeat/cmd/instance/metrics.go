// +build darwin,cgo freebsd,cgo linux windows

package instance

import (
	"fmt"
	"runtime"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/cpu"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/beats/libbeat/monitoring"
)

var (
	beatProcessStats *process.Stats
	systemMetrics    *monitoring.Registry
)

func init() {
	systemMetrics = monitoring.Default.NewRegistry("system")
}

func setupMetrics(name string) error {
	monitoring.NewFunc(beatMetrics, "memstats", reportMemStats, monitoring.Report)
	monitoring.NewFunc(beatMetrics, "cpu", reportBeatCPU, monitoring.Report)

	monitoring.NewFunc(systemMetrics, "cpu", reportSystemCPUUsage, monitoring.Report)
	if runtime.GOOS != "windows" {
		monitoring.NewFunc(systemMetrics, "load", reportSystemLoadAverage, monitoring.Report)
	}

	beatProcessStats = &process.Stats{
		Procs:        []string{name},
		EnvWhitelist: nil,
		CpuTicks:     true,
		CacheCmdLine: true,
		IncludeTop:   process.IncludeTopConfig{},
	}
	err := beatProcessStats.Init()

	return err
}

func reportMemStats(m monitoring.Mode, V monitoring.Visitor) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	monitoring.ReportInt(V, "memory_total", int64(stats.TotalAlloc))
	if m == monitoring.Full {
		monitoring.ReportInt(V, "memory_alloc", int64(stats.Alloc))
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
	pid, err := process.GetSelfPid()
	if err != nil {
		return 0, fmt.Errorf("error getting PID for self process: %v", err)
	}

	state, err := beatProcessStats.GetOne(pid)
	if err != nil {
		return 0, fmt.Errorf("error retrieving process stats: %v", err)
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
	pid, err := process.GetSelfPid()
	if err != nil {
		return 0.0, nil, fmt.Errorf("error getting PID for self process: %v", err)
	}

	state, err := beatProcessStats.GetOne(pid)
	if err != nil {
		return 0.0, nil, fmt.Errorf("error retrieving process stats: %v", err)
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

	monitoring.ReportInt(V, "cores", int64(process.NumCPU))
}
