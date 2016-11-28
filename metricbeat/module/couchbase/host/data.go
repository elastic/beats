package host

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"io"
)

type StorageTotals_Ram struct {
	Total             int64 `json:"total"`
	QuotaTotal        int64 `json:"quotaTotal"`
	QuotaUsed         int `json:"quotaUsed"`
	Used              int64 `json:"used"`
	UsedByData        int `json:"usedByData"`
	QuotaUsedPerNode  int `json:"quotaUsedPerNode"`
	QuotaTotalPerNode int64 `json:"quotaTotalPerNode"`
}
type StorageTotals_Hdd struct {
	Total      int64 `json:"total"`
	QuotaTotal int64 `json:"quotaTotal"`
	Used       int64 `json:"used"`
	UsedByData int `json:"usedByData"`
	Free       int64 `json:"free"`
}

type StorageTotals struct {
	RAM StorageTotals_Ram `json:"ram"`
	Hdd StorageTotals_Hdd `json:"hdd"`
}

type NodeSystemStats struct {
	CPUUtilizationRate int `json:"cpu_utilization_rate"`
	SwapTotal          int64 `json:"swap_total"`
	SwapUsed           int `json:"swap_used"`
	MemTotal           int64 `json:"mem_total"`
	MemFree            int64 `json:"mem_free"`
}

type NodeInterestingStats struct {
	CmdGet                   int `json:"cmd_get"`
	CouchDocsActualDiskSize  int `json:"couch_docs_actual_disk_size"`
	CouchDocsDataSize        int `json:"couch_docs_data_size"`
	CouchSpatialDataSize     int `json:"couch_spatial_data_size"`
	CouchSpatialDiskSize     int `json:"couch_spatial_disk_size"`
	CouchViewsActualDiskSize int `json:"couch_views_actual_disk_size"`
	CouchViewsDataSize       int `json:"couch_views_data_size"`
	CurrItems                int `json:"curr_items"`
	CurrItemsTot             int `json:"curr_items_tot"`
	EpBgFetched              int `json:"ep_bg_fetched"`
	GetHits                  int `json:"get_hits"`
	MemUsed                  int `json:"mem_used"`
	Ops                      int `json:"ops"`
	VbReplicaCurrItems       int `json:"vb_replica_curr_items"`
}

type Node struct {
	SystemStats        NodeSystemStats `json:"systemStats"`
	InterestingStats   NodeInterestingStats `json:"interestingStats"`
	Uptime             string `json:"uptime"`
	MemoryTotal        int64 `json:"memoryTotal"`
	MemoryFree         int64 `json:"memoryFree"`
	McdMemoryReserved  int `json:"mcdMemoryReserved"`
	McdMemoryAllocated int `json:"mcdMemoryAllocated"`
	ClusterMembership  string `json:"clusterMembership"`
	RecoveryType       string `json:"recoveryType"`
	Status             string `json:"status"`
	ThisNode           bool `json:"thisNode"`
	Hostname           string `json:"hostname"`
	ClusterCompatibility int `json:"clusterCompatibility"`
	Version              string `json:"version"`
	Os                   string `json:"os"`
}

type Data struct {
	StorageTotals        StorageTotals `json:"storageTotals"`
	IndexMemoryQuota     int `json:"indexMemoryQuota"`
	MemoryQuota          int `json:"memoryQuota"`
	Name                 string `json:"name"`
	Nodes                []Node `json:"nodes"`
	RebalanceStatus      string `json:"rebalanceStatus"`
	RebalanceProgressURI string `json:"rebalanceProgressUri"`
	StopRebalanceURI     string `json:"stopRebalanceUri"`
	NodeStatusesURI      string `json:"nodeStatusesUri"`
	MaxBucketCount       int `json:"maxBucketCount"`
}

func eventsMapping(body io.Reader, hostname string) common.MapStr {

	var d Data
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	logp.Info("Printing Data:")
	hostEvent := common.MapStr{
		"hostname": hostname,
		"name": d.Name,
		"indexMemoryQuota": d.IndexMemoryQuota,
		"maxBucketCount": d.MaxBucketCount,
		"memoryQuota": d.MemoryQuota,
		"storage_totals": common.MapStr{
			"hdd": common.MapStr{
				"free": d.StorageTotals.Hdd.Free,
				"quotaTotal": d.StorageTotals.Hdd.QuotaTotal,
				"total": d.StorageTotals.Hdd.Total,
				"used": d.StorageTotals.Hdd.Used,
				"usedByData": d.StorageTotals.Hdd.UsedByData,

			},
			"ram": common.MapStr{
				"quotaTotal": d.StorageTotals.RAM.QuotaTotal,
				"quotaTotalPerNode": d.StorageTotals.RAM.QuotaTotalPerNode,
				"quotaUsed": d.StorageTotals.RAM.QuotaUsed,
				"quotaUsedPerNode": d.StorageTotals.RAM.QuotaUsedPerNode,
				"total": d.StorageTotals.RAM.Total,
				"used": d.StorageTotals.RAM.Used,
				"usedByData": d.StorageTotals.RAM.UsedByData,
			},

		},
	}

	return hostEvent
}

