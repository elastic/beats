package memory

import (
	"github.com/elastic/beats/libbeat/common"
)

func eventsMapping(memoryDataList []MemoryData) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, memoryData := range memoryDataList {
		myEvents = append(myEvents, eventMapping(&memoryData))
	}
	return myEvents
}
func eventMapping(memoryData *MemoryData) common.MapStr {

	event := common.MapStr{
		"@timestamp": memoryData.Time,
		"container": common.MapStr{
			"id":     memoryData.MyContainer.Id,
			"name":   memoryData.MyContainer.Name,
			"labels": memoryData.MyContainer.Labels,
		},
		"memory": common.MapStr{
			"failcnt":    memoryData.Failcnt,
			"limit":      memoryData.Limit,
			"maxUsage":   memoryData.MaxUsage,
			"totalRss":   memoryData.TotalRss,
			"totalRss_p": memoryData.TotalRss_p,
			"usage":      memoryData.Usage,
			"usage_p":    memoryData.Usage_p,
		},
	}
	return event
}
