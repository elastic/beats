package network

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/docker"
)

func eventsMapping(netsStatsList []NETstats) []common.MapStr {
	myEvents := []common.MapStr{}
	for _, netsStats := range netsStatsList {
		myEvents = append(myEvents, eventMapping(&netsStats))
	}
	return myEvents
}
func eventMapping(myNetStats *NETstats) common.MapStr {
	event := common.MapStr{
		"@timestamp": myNetStats.Time,
		"container": common.MapStr{
			"id":     myNetStats.MyContainer.Id,
			"name":   myNetStats.MyContainer.Name,
			"labels": myNetStats.MyContainer.Labels,
		},
		"socket": docker.GetSocket(),
		myNetStats.NameInterface: common.MapStr{
			"rx": common.MapStr{
				"bytes":   myNetStats.RxBytes,
				"dropped": myNetStats.RxDropped,
				"errors":  myNetStats.RxErrors,
				"packets": myNetStats.RxPackets,
			},
			"tx": common.MapStr{
				"bytes":   myNetStats.TxBytes,
				"dropped": myNetStats.TxDropped,
				"errors":  myNetStats.TxErrors,
				"packets": myNetStats.TxPackets,
			},
		},
	}
	return event
}
