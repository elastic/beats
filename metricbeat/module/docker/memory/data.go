package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
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
		"socket": docker.GetSocket(),
		"memory": common.MapStr{
			"failcnt":     memoryData.Failcnt,
			"limit":       memoryData.Limit,
			"max_usage":   memoryData.MaxUsage,
			"total_rss":   memoryData.TotalRss,
			"total_rss_p": memoryData.TotalRss_p,
			"usage":       memoryData.Usage,
			"usage_p":     memoryData.Usage_p,
		},
	}
	return event
}
