package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

func eventsMapping(memoryDataList []MemoryData) []common.MapStr {
	events := []common.MapStr{}
	for _, memoryData := range memoryDataList {
		events = append(events, eventMapping(&memoryData))
	}
	return events
}

func eventMapping(memoryData *MemoryData) common.MapStr {
	event := common.MapStr{
		mb.ModuleDataKey: common.MapStr{
			"container": memoryData.Container.ToMapStr(),
		},
		"fail": common.MapStr{
			"count": memoryData.Failcnt,
		},
		"limit": memoryData.Limit,
		"rss": common.MapStr{
			"total": memoryData.TotalRss,
			"pct":   memoryData.TotalRssP,
		},
		"usage": common.MapStr{
			"total": memoryData.Usage,
			"pct":   memoryData.UsageP,
			"max":   memoryData.MaxUsage,
		},
	}
	return event
}
