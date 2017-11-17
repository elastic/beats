package cluster

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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

func eventMapping(content []byte) common.MapStr {
	var d Data
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	logp.Info("Printing Data:")
	event := common.MapStr{
		"hdd": common.MapStr{
			"quota": common.MapStr{
				"total": common.MapStr{
					"bytes": d.StorageTotals.Hdd.QuotaTotal,
				},
			},
			"free": common.MapStr{
				"bytes": d.StorageTotals.Hdd.Free,
			},
			"total": common.MapStr{
				"bytes": d.StorageTotals.Hdd.Total,
			},
			"used": common.MapStr{
				"value": common.MapStr{
					"bytes": d.StorageTotals.Hdd.Used,
				},
				"by_data": common.MapStr{
					"bytes": d.StorageTotals.Hdd.UsedByData,
				},
			},
		},
		"max_bucket_count": d.MaxBucketCount,
		"quota": common.MapStr{
			"index_memory": common.MapStr{
				"mb": d.IndexMemoryQuota,
			},
			"memory": common.MapStr{
				"mb": d.MemoryQuota,
			},
		},
		"ram": common.MapStr{
			"quota": common.MapStr{
				"total": common.MapStr{
					"value": common.MapStr{
						"bytes": d.StorageTotals.RAM.QuotaTotal,
					},
					"per_node": common.MapStr{
						"bytes": d.StorageTotals.RAM.QuotaTotalPerNode,
					},
				},
				"used": common.MapStr{
					"value": common.MapStr{
						"bytes": d.StorageTotals.RAM.QuotaUsed,
					},
					"per_node": common.MapStr{
						"bytes": d.StorageTotals.RAM.QuotaUsedPerNode,
					},
				},
			},
			"total": common.MapStr{
				"bytes": d.StorageTotals.RAM.Total,
			},
			"used": common.MapStr{
				"value": common.MapStr{
					"bytes": d.StorageTotals.RAM.Used,
				},
				"by_data": common.MapStr{
					"bytes": d.StorageTotals.RAM.UsedByData,
				},
			},
		},
	}

	return event
}
