package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(cpuStatsList []CPUStats) []common.MapStr {
	events := []common.MapStr{}
	for _, cpuStats := range cpuStatsList {
		events = append(events, eventMapping(&cpuStats))
	}
	return events
}

func eventMapping(stats *CPUStats) common.MapStr {
	event := common.MapStr{
		mb.ModuleDataKey: common.MapStr{
			"container": stats.Container.ToMapStr(),
		},
		"core": stats.PerCpuUsage,
		"total": common.MapStr{
			"pct": stats.TotalUsage,
		},
		"kernel": common.MapStr{
			"ticks": stats.UsageInKernelmode,
			"pct":   stats.UsageInKernelmodePercentage,
		},
		"user": common.MapStr{
			"ticks": stats.UsageInUsermode,
			"pct":   stats.UsageInUsermodePercentage,
		},
		"system": common.MapStr{
			"ticks": stats.SystemUsage,
			"pct":   stats.SystemUsagePercentage,
		},
	}

	return event
}
