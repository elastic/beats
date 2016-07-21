package memory
import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
	"github.com/elastic/beats/metricbeat/module/docker/services/config"
)

func eventsMapping( memoryDatas []config.MEMORYData) [] common.MapStr {
	myEvents := [] common.MapStr{}
	for _, memoryData := range memoryDatas {
		myEvents = append(myEvents, eventMapping(memoryData))
	}
	return myEvents
}
func eventMapping(memoryData config.MEMORYData) common.MapStr{

	event := common.MapStr{
		"@timestamp":	memoryData.MyContainer.Time,
		"type": "memory",
		"container": 	common.MapStr{
			"id":  memoryData.MyContainer.Id,
			"name": memoryData.MyContainer.Name,
			"labels": memoryData.MyContainer.Labels,
		},
		"dockerSocket": memoryData.MyContainer.Socket,
		"memory": common.MapStr{
			"failcnt":    memoryData.Failcnt,
			"limit":      memoryData.Limit,
			"maxUsage":   memoryData.MaxUsage,
			"totalRss":   memoryData.TotalRss,
			"totalRss_p": memoryData.TotalRss_p,
			"usage":      memoryData.Usage,
			"usage_p":   memoryData.Usage_p,
		},
	}
	return event
}
