package instance

import (
	"runtime"
	"time"

	"github.com/elastic/beats/libbeat/monitoring"
)

func init() {
	metrics := monitoring.Default.NewRegistry("beat")

	monitoring.NewFunc(metrics, "memstats", reportMemStats, monitoring.Report)
	monitoring.NewFunc(metrics, "info", reportInfo, monitoring.Report)
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

	delta := time.Since(startTime)
	uptime := int64(delta / time.Millisecond)
	monitoring.ReportInt(V, "uptime.ms", uptime)
}
