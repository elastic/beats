package memory

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/syncgateway"
)

func eventMapping(r mb.ReporterV2, content *syncgateway.SgResponse) {
	delete(content.MemStats, "BySize")
	delete(content.MemStats, "PauseNs")
	delete(content.MemStats, "PauseEnd")

	r.Event(mb.Event{
		MetricSetFields: content.MemStats,
	})
}
