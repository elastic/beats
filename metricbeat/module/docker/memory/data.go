package memory

import "github.com/elastic/beats/libbeat/common"

func eventsMapping(memoryDataList []MemoryData) []common.MapStr {
	events := []common.MapStr{}
	for _, memoryData := range memoryDataList {
		events = append(events, eventMapping(&memoryData))
	}
	return events
}

func eventMapping(memoryData *MemoryData) common.MapStr {

	event := common.MapStr{
		"container":  memoryData.Container.ToMapStr(),
		"fail.count": memoryData.Failcnt,
		"limit":      memoryData.Limit,
		"total": common.MapStr{
			"rss":     memoryData.TotalRss,
			"rss.pct": memoryData.TotalRss_p,
		},
		"usage": common.MapStr{
			"total": memoryData.Usage,
			"pct":   memoryData.Usage_p,
			"max":   memoryData.MaxUsage,
		},
	}
	return event
}
