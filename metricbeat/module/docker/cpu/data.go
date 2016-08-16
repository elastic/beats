package cpu

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
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
		"socket": docker.GetSocket(),
		"cpu": common.MapStr{
			"per_cpu_usage":        mycpuStats.PerCpuUsage,
			"total_usage":          mycpuStats.TotalUsage,
			"usage_in_kernel_mode": mycpuStats.UsageInKernelmode,
			"usage_in_user_mode":   mycpuStats.UsageInUsermode,
		},
	}
	return event
}
