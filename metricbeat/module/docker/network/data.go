package network

import (
	"github.com/elastic/beats/libbeat/common"
	//"fmt"
)

func eventsMapping(netsStatsList []NETstats) []common.MapStr {
	myEvents := []common.MapStr{}
	//fmt.Printf(" Taille cpuStatsList : ",len(cpuStatsList),"\n")
	for _, netsStats := range netsStatsList {
		myEvents = append(myEvents, eventMapping(&netsStats))
	}
	return myEvents
}
func eventMapping(myNetStats *NETstats) common.MapStr {
	event := common.MapStr{
		"@timestamp": myNetStats.Time,
		"type":       "net",
		"container": common.MapStr{
			"id":     myNetStats.MyContainer.Id,
			"name":   myNetStats.MyContainer.Name,
			"labels": myNetStats.MyContainer.Labels,
		},
		//"dockerSocket": myNetStats.MyContainer.Socket,
		myNetStats.NameInterface: common.MapStr{
			"rxBytes":   myNetStats.RxBytes,
			"rxDropped": myNetStats.RxDropped,
			"rxErrors":  myNetStats.RxErrors,
			"rxPackets": myNetStats.RxPackets,
			"txBytes":   myNetStats.TxBytes,
			"txDropped": myNetStats.TxDropped,
			"txErrors":  myNetStats.TxErrors,
			"txPackets": myNetStats.TxPackets,
		},
	}
	return event
}
