// +build darwin linux windows
// +build cgo

package instance

import (
	"fmt"
	"runtime"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/cpu"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/report/log"
)

var (
	cpuMonitor       *cpu.Monitor
	beatProcessStats *process.Stats
)

func init() {
	beatMetrics := monitoring.Default.NewRegistry("beat")
	monitoring.NewFunc(beatMetrics, "memstats", reportMemStats, monitoring.Report)
	monitoring.NewFunc(beatMetrics, "cpu", reportBeatCPU, monitoring.Report)
	monitoring.NewFunc(beatMetrics, "info", reportInfo, monitoring.Report)

	systemMetrics := monitoring.Default.NewRegistry("system")
	monitoring.NewFunc(systemMetrics, "load", reportSystemLoadAverage, monitoring.Report)
	monitoring.NewFunc(systemMetrics, "cpu", reportSystemCPUUsage, monitoring.Report)
}

func setupMetrics(name string) error {
	cpuMonitor = new(cpu.Monitor)

	beatProcessStats = &process.Stats{
		Procs:        []string{name},
		EnvWhitelist: nil,
		CpuTicks:     false,
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
}

func reportInfo(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	delta := time.Since(log.StartTime)
	uptime := int64(delta / time.Millisecond)
	monitoring.ReportInt(V, "uptime.ms", uptime)
}

func reportBeatCPU(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	cpuUsage, cpuUsageNorm, totalCPUUsage, err := getCPUPercentages()
	if err != nil {
		logp.Err("Error retrieving CPU percentages: %v", err)
		return
	}
	monitoring.ReportFloat(V, "total.pct", cpuUsage)
	monitoring.ReportFloat(V, "total.norm.pct", cpuUsageNorm)
	monitoring.ReportFloat(V, "total.value", totalCPUUsage)

}

func getCPUPercentages() (float64, float64, float64, error) {
	state, err := beatProcessStats.Get()
	if err != nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("error retrieving process stats")
	}

	if len(state) != 1 {
		return 0.0, 0.0, 0.0, fmt.Errorf("more or less than one processes: %v", len(state))
	}

	beatState := state[0]
	iCPUUsage, err := beatState.GetValue("cpu.total.pct")
	if err != nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("error getting total CPU usage: %v", err)
	}
	iCPUUsageNorm, err := beatState.GetValue("cpu.total.norm.pct")
	if err != nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("error getting normalized CPU percentage: %v", err)
	}

	iTotalCPUUsage, err := beatState.GetValue("cpu.total.value")
	if err != nil {
		return 0.0, 0.0, 0.0, fmt.Errorf("error getting total CPU since start: %v", err)
	}

	cpuUsage, ok := iCPUUsage.(float64)
	if !ok {
		return 0.0, 0.0, 0.0, fmt.Errorf("error converting value of CPU usage")
	}

	cpuUsageNorm, ok := iCPUUsageNorm.(float64)
	if !ok {
		return 0.0, 0.0, 0.0, fmt.Errorf("error converting value of normalized CPU usage")
	}

	totalCPUUsage, ok := iTotalCPUUsage.(float64)
	if !ok {
		return 0.0, 0.0, 0.0, fmt.Errorf("error converting value of CPU usage since start")
	}

	return cpuUsage, cpuUsageNorm, totalCPUUsage, nil
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
	monitoring.ReportFloat(V, "norm.1", normAvgs.OneMinute)
	monitoring.ReportFloat(V, "norm.5", normAvgs.FiveMinute)
	monitoring.ReportFloat(V, "norm.15", normAvgs.FifteenMinute)
}

func reportSystemCPUUsage(_ monitoring.Mode, V monitoring.Visitor) {
	V.OnRegistryStart()
	defer V.OnRegistryFinished()

	sample, err := cpuMonitor.Sample()
	if err != nil {
		logp.Err("Error retrieving CPU usage of the host: %v", err)
		return
	}

	pct := sample.Percentages()
	monitoring.ReportFloat(V, "total.pct", pct.Total)

	normalizedPct := sample.NormalizedPercentages()
	monitoring.ReportFloat(V, "total.norm.pct", normalizedPct.Total)

	monitoring.ReportInt(V, "cores", int64(process.NumCPU))
}
