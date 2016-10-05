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
		mb.ModuleData: common.MapStr{
			"container": stats.Container.ToMapStr(),
		},
		"usage": common.MapStr{
			"per_cpu":     stats.PerCpuUsage,
			"total":       stats.TotalUsage,
			"kernel_mode": stats.UsageInKernelmode,
			"user_mode":   stats.UsageInUsermode,
		},
	}

	return event
}
