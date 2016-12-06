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
	IndexMemoryQuota     int64         `json:"indexMemoryQuota"`
	MemoryQuota          int64         `json:"memoryQuota"`
	RebalanceStatus      string        `json:"rebalanceStatus"`
	RebalanceProgressURI string        `json:"rebalanceProgressUri"`
	StopRebalanceURI     string        `json:"stopRebalanceUri"`
	NodeStatusesURI      string        `json:"nodeStatusesUri"`
	MaxBucketCount       int64         `json:"maxBucketCount"`
}

func eventMapping(body io.Reader) common.MapStr {

	var d Data
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	logp.Info("Printing Data:")
	event := common.MapStr{
		"quota.index_memory.mb":          d.IndexMemoryQuota,
		"quota.memory.mb":                d.MemoryQuota,
		"max_bucket_count":               d.MaxBucketCount,
		"hdd.free.bytes":                 d.StorageTotals.Hdd.Free,
		"hdd.quota_total.bytes":          d.StorageTotals.Hdd.QuotaTotal,
		"hdd.total.bytes":                d.StorageTotals.Hdd.Total,
		"hdd.used.bytes":                 d.StorageTotals.Hdd.Used,
		"hdd.used.by_data.bytes":         d.StorageTotals.Hdd.UsedByData,
		"ram.quota.total.bytes":          d.StorageTotals.RAM.QuotaTotal,
		"ram.quota.total.per_node.bytes": d.StorageTotals.RAM.QuotaTotalPerNode,
		"ram.quota.used.bytes":           d.StorageTotals.RAM.QuotaUsed,
		"ram.quota.used.per_node.bytes":  d.StorageTotals.RAM.QuotaUsedPerNode,
		"ram.total.bytes":                d.StorageTotals.RAM.Total,
		"ram.used.bytes":                 d.StorageTotals.RAM.Used,
		"ram.used.by_data.bytes":         d.StorageTotals.RAM.UsedByData,
	}

	return event
}
