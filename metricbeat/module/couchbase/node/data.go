package node

import (
	"encoding/json"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type NodeSystemStats struct {
	CPUUtilizationRate float32 `json:"cpu_utilization_rate"`
	SwapTotal          int64   `json:"swap_total"`
	SwapUsed           int64   `json:"swap_used"`
	MemTotal           int64   `json:"mem_total"`
	MemFree            int64   `json:"mem_free"`
}

type NodeInterestingStats struct {
	CmdGet                   int64 `json:"cmd_get"`
	CouchDocsActualDiskSize  int64 `json:"couch_docs_actual_disk_size"`
	CouchDocsDataSize        int64 `json:"couch_docs_data_size"`
	CouchSpatialDataSize     int64 `json:"couch_spatial_data_size"`
	CouchSpatialDiskSize     int64 `json:"couch_spatial_disk_size"`
	CouchViewsActualDiskSize int64 `json:"couch_views_actual_disk_size"`
	CouchViewsDataSize       int64 `json:"couch_views_data_size"`
	CurrItems                int64 `json:"curr_items"`
	CurrItemsTot             int64 `json:"curr_items_tot"`
	EpBgFetched              int64 `json:"ep_bg_fetched"`
	GetHits                  int64 `json:"get_hits"`
	MemUsed                  int64 `json:"mem_used"`
	Ops                      int64 `json:"ops"`
	VbReplicaCurrItems       int64 `json:"vb_replica_curr_items"`
}

type Node struct {
	SystemStats          NodeSystemStats      `json:"systemStats"`
	InterestingStats     NodeInterestingStats `json:"interestingStats"`
	Uptime               string               `json:"uptime"`
	MemoryTotal          int64                `json:"memoryTotal"`
	MemoryFree           int64                `json:"memoryFree"`
	McdMemoryReserved    int64                `json:"mcdMemoryReserved"`
	McdMemoryAllocated   int64                `json:"mcdMemoryAllocated"`
	ClusterMembership    string               `json:"clusterMembership"`
	RecoveryType         string               `json:"recoveryType"`
	Status               string               `json:"status"`
	ThisNode             bool                 `json:"thisNode"`
	Hostname             string               `json:"hostname"`
	ClusterCompatibility int64                `json:"clusterCompatibility"`
	Version              string               `json:"version"`
	Os                   string               `json:"os"`
}

type Data struct {
	Nodes []Node `json:"nodes"`
}

func eventsMapping(body io.Reader) []common.MapStr {

	var d Data
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	for _, NodeItem := range d.Nodes {
		event := common.MapStr{
			"hostname":                           NodeItem.Hostname,
			"uptime":                             NodeItem.Uptime,
			"mcd_memory.reserved.bytes":          NodeItem.McdMemoryReserved,
			"mcd_memory.allocated.bytes":         NodeItem.McdMemoryAllocated,
			"cmd_get":                            NodeItem.InterestingStats.CmdGet,
			"cpu_utilization_rate.pct":           NodeItem.SystemStats.CPUUtilizationRate,
			"couch_docs_actual_disk_size.bytes":  NodeItem.InterestingStats.CouchDocsActualDiskSize,
			"couch_docs_data_size.bytes":         NodeItem.InterestingStats.CouchDocsDataSize,
			"couch_spatial_data_size.bytes":      NodeItem.InterestingStats.CouchSpatialDataSize,
			"couch_spatial_disk_size.bytes":      NodeItem.InterestingStats.CouchSpatialDiskSize,
			"couch_views_actual_disk_size.bytes": NodeItem.InterestingStats.CouchViewsActualDiskSize,
			"couch_views_data_size.bytes":        NodeItem.InterestingStats.CouchViewsDataSize,
			"curr_items":                         NodeItem.InterestingStats.CurrItems,
			"curr_items_tot":                     NodeItem.InterestingStats.CurrItemsTot,
			"ep_bg_fetched":                      NodeItem.InterestingStats.EpBgFetched,
			"get_hits":                           NodeItem.InterestingStats.GetHits,
			"ops":                                NodeItem.InterestingStats.Ops,
			"vb_replica_curr_items": NodeItem.InterestingStats.VbReplicaCurrItems,
			"swap.total.bytes":      NodeItem.SystemStats.SwapTotal,
			"swap.used.bytes":       NodeItem.SystemStats.SwapUsed,
			"mem.total.bytes":       NodeItem.SystemStats.MemTotal,
			"mem.free.bytes":        NodeItem.SystemStats.MemFree,
			"mem.used.bytes":        NodeItem.InterestingStats.MemUsed,
		}
		events = append(events, event)
	}

	return events

}
