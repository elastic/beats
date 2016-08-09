package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
)

func eventsMapping(cpuStatsList []CPUStats) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, cpuStats := range cpuStatsList {
		myEvents = append(myEvents, eventMapping(&cpuStats))
	}
	return myEvents
}
func eventMapping(mycpuStats *CPUStats) common.MapStr {

	event := common.MapStr{
		"@timestamp": mycpuStats.Time,
		"container": common.MapStr{
			"id":     mycpuStats.MyContainer.Id,
			"name":   mycpuStats.MyContainer.Name,
			"labels": mycpuStats.MyContainer.Labels,
		},
		"cpu": common.MapStr{
			"percpuUsage":       mycpuStats.PerCpuUsage,
			"totalUsage":        mycpuStats.TotalUsage,
			"usageInKernelmode": mycpuStats.UsageInKernelmode,
			"usageInUsermode":   mycpuStats.UsageInUsermode,
		},
	}
	return event
}
