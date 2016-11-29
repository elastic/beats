package cluster

import (
	"encoding/json"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"io"
)

type StorageTotals_Ram struct {
	Total             int64 `json:"total"`
	QuotaTotal        int64 `json:"quotaTotal"`
	QuotaUsed         int64 `json:"quotaUsed"`
	Used              int64 `json:"used"`
	UsedByData        int64 `json:"usedByData"`
	QuotaUsedPerNode  int64 `json:"quotaUsedPerNode"`
	QuotaTotalPerNode int64 `json:"quotaTotalPerNode"`
}
type StorageTotals_Hdd struct {
	Total      int64 `json:"total"`
	QuotaTotal int64 `json:"quotaTotal"`
	Used       int64 `json:"used"`
	UsedByData int64 `json:"usedByData"`
	Free       int64 `json:"free"`
}

type StorageTotals struct {
	RAM StorageTotals_Ram `json:"ram"`
	Hdd StorageTotals_Hdd `json:"hdd"`
}

type Data struct {
	StorageTotals        StorageTotals `json:"storageTotals"`
	IndexMemoryQuota     int64 `json:"indexMemoryQuota"`
	MemoryQuota          int64 `json:"memoryQuota"`
	RebalanceStatus      string `json:"rebalanceStatus"`
	RebalanceProgressURI string `json:"rebalanceProgressUri"`
	StopRebalanceURI     string `json:"stopRebalanceUri"`
	NodeStatusesURI      string `json:"nodeStatusesUri"`
	MaxBucketCount       int64 `json:"maxBucketCount"`
}

func eventMapping(body io.Reader) common.MapStr {

	var d Data
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	logp.Info("Printing Data:")
	event := common.MapStr{
		"indexMemoryQuota": d.IndexMemoryQuota,
		"maxBucketCount": d.MaxBucketCount,
		"memoryQuota": d.MemoryQuota,
		"hdd_free": d.StorageTotals.Hdd.Free,
		"hdd_quotaTotal": d.StorageTotals.Hdd.QuotaTotal,
		"hdd_total": d.StorageTotals.Hdd.Total,
		"hdd_used": d.StorageTotals.Hdd.Used,
		"hdd_usedByData": d.StorageTotals.Hdd.UsedByData,
		"ram_quotaTotal": d.StorageTotals.RAM.QuotaTotal,
		"ram_quotaTotalPerNode": d.StorageTotals.RAM.QuotaTotalPerNode,
		"ram_quotaUsed": d.StorageTotals.RAM.QuotaUsed,
		"ram_quotaUsedPerNode": d.StorageTotals.RAM.QuotaUsedPerNode,
		"ram_total": d.StorageTotals.RAM.Total,
		"ram_used": d.StorageTotals.RAM.Used,
		"ram_usedByData": d.StorageTotals.RAM.UsedByData,
	}

	return event
}
