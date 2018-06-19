package network

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(r mb.ReporterV2, netsStatsList []NetStats) {
	for _, netsStats := range netsStatsList {
		eventMapping(r, &netsStats)
	}
}

func eventMapping(r mb.ReporterV2, stats *NetStats) {
	// Deprecated fields
	r.Event(mb.Event{
		ModuleFields: common.MapStr{
			"container": stats.Container.ToMapStr(),
		},
		MetricSetFields: common.MapStr{
			"interface": stats.NameInterface,
			"in": common.MapStr{
				"bytes":   stats.RxBytes,
				"dropped": stats.RxDropped,
				"errors":  stats.RxErrors,
				"packets": stats.RxPackets,
			},
			"out": common.MapStr{
				"bytes":   stats.TxBytes,
				"dropped": stats.TxDropped,
				"errors":  stats.TxErrors,
				"packets": stats.TxPackets,
			},
			"inbound": common.MapStr{
				"bytes":   stats.Total.RxBytes,
				"dropped": stats.Total.RxDropped,
				"errors":  stats.Total.RxErrors,
				"packets": stats.Total.RxPackets,
			},
			"outbound": common.MapStr{
				"bytes":   stats.Total.TxBytes,
				"dropped": stats.Total.TxDropped,
				"errors":  stats.Total.TxErrors,
				"packets": stats.Total.TxPackets,
			},
		},
	})
}
