package cpu
import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
)

func eventsMapping( cpuStatsList []CPUStats) [] common.MapStr {
	myEvents := [] common.MapStr{}
	//fmt.Printf(" Taille cpuStatsList : ",len(cpuStatsList),"\n")
	for _, cpuStats := range cpuStatsList {
		myEvents = append(myEvents, eventMapping(&cpuStats))
	}
	//fmt.Printf(" Taille events : ",len(myEvents),"\n")
	return myEvents
}
func eventMapping( mycpuStats *CPUStats) common.MapStr{

	//fmt.Printf(" From data : Nom du container : ",mycpuStats.MyContainer.Name,"\n")
	//fmt.Printf(" ID du container : ",mycpuStats.MyContainer.Id,"\n")

	event := common.MapStr{
		"@timestamp":      mycpuStats.Time,
		"type":            "cpu",
		"container": common.MapStr{
			"id":  mycpuStats.MyContainer.Id,
			"name": mycpuStats.MyContainer.Name,
			"labels": mycpuStats.MyContainer.Labels,
		},
		//"dockerSocket": mycpuStats.MyContainer.Socket,
		"cpu": common.MapStr{
			"percpuUsage":       mycpuStats.PerCpuUsage,
			"totalUsage":        mycpuStats.TotalUsage,
			"usageInKernelmode": mycpuStats.UsageInKernelmode,
			"usageInUsermode":   mycpuStats.UsageInUsermode,
		},
	}
	return event
}
