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
			"rx_bytes":   myNetStats.RxBytes,
			"rx_dropped": myNetStats.RxDropped,
			"rx_errors":  myNetStats.RxErrors,
			"rx_packets": myNetStats.RxPackets,
			"tx_bytes":   myNetStats.TxBytes,
			"tx_dropped": myNetStats.TxDropped,
			"tx_errors":  myNetStats.TxErrors,
			"tx_packets": myNetStats.TxPackets,
		},
	}
	return event
}
