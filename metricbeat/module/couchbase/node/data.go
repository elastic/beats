package node

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"io"
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
			"hostname":                 NodeItem.Hostname,
			"uptime":                   NodeItem.Uptime,
			"memoryTotal":              NodeItem.MemoryTotal,
			"memoryFree":               NodeItem.MemoryFree,
			"mcdMemoryReserved":        NodeItem.McdMemoryReserved,
			"mcdMemoryAllocated":       NodeItem.McdMemoryAllocated,
			"cmdGet":                   NodeItem.InterestingStats.CmdGet,
			"couchDocsActualDiskSize":  NodeItem.InterestingStats.CouchDocsActualDiskSize,
			"couchDocsDataSize":        NodeItem.InterestingStats.CouchDocsDataSize,
			"couchSpatialDataSize":     NodeItem.InterestingStats.CouchSpatialDataSize,
			"couchSpatialDiskSize":     NodeItem.InterestingStats.CouchSpatialDiskSize,
			"couchViewsActualDiskSize": NodeItem.InterestingStats.CouchViewsActualDiskSize,
			"couchViewsDataSize":       NodeItem.InterestingStats.CouchViewsDataSize,
			"currItems":                NodeItem.InterestingStats.CurrItems,
			"currItemsTot":             NodeItem.InterestingStats.CurrItemsTot,
			"epBgFetched":              NodeItem.InterestingStats.EpBgFetched,
			"getHits":                  NodeItem.InterestingStats.GetHits,
			"memUsed":                  NodeItem.InterestingStats.MemUsed,
			"ops":                      NodeItem.InterestingStats.Ops,
			"vbReplicaCurrItems":       NodeItem.InterestingStats.VbReplicaCurrItems,
			"CPUUtilizationRate":       NodeItem.SystemStats.CPUUtilizationRate,
			"swapTotal":                NodeItem.SystemStats.SwapTotal,
			"swapUsed":                 NodeItem.SystemStats.SwapUsed,
			"memTotal":                 NodeItem.SystemStats.MemTotal,
			"memFree":                  NodeItem.SystemStats.MemFree,
		}
		events = append(events, event)
	}

	return events

}
