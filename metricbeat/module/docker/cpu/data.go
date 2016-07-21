package cpu
import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
	"github.com/elastic/beats/metricbeat/module/docker/services/config"
)

func eventsMapping( cpuDatas []config.CPUData) [] common.MapStr {
	myEvents := [] common.MapStr{}
	//fmt.Printf(" Taille est  CPUDATA:" ,len(cpuDatas))
	for _, cpuData := range cpuDatas {
		myEvents = append(myEvents, eventMapping(cpuData))
	}
	return myEvents
}
func eventMapping(cpuData config.CPUData) common.MapStr{

	event := common.MapStr{
		"@timestamp":      cpuData.MyContainer.Time,
		"type":            "cpu",
		"container": common.MapStr{
			"id":  cpuData.MyContainer.Id,
			"name": cpuData.MyContainer.Name,
			"labels": cpuData.MyContainer.Labels,
		},
		"dockerSocket": cpuData.MyContainer.Socket,
		"cpu": common.MapStr{
			"percpuUsage":       cpuData.PerCpuUsage,
			"totalUsage":        cpuData.TotalUsage,
			"usageInKernelmode": cpuData.UsageInKernelmode,
			"usageInUsermode":   cpuData.UsageInUsermode,
		},
	}
	return event
}
