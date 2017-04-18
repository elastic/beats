package beat

import (
	"runtime"

	"github.com/elastic/beats/libbeat/monitoring"
)

type memstatsVar struct{}

var (
	metrics = monitoring.Default.NewRegistry("beat")
)

func init() {

	var ms memstatsVar
	metrics.Add("memstats", ms, monitoring.Reported)
}

func (memstatsVar) Visit(m monitoring.Mode, V monitoring.Visitor) {
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
