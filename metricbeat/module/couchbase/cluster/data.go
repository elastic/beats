// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cluster

import (
	"encoding/json"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	IndexMemoryQuota     float64       `json:"indexMemoryQuota"`
	MemoryQuota          float64       `json:"memoryQuota"`
	RebalanceStatus      string        `json:"rebalanceStatus"`
	RebalanceProgressURI string        `json:"rebalanceProgressUri"`
	StopRebalanceURI     string        `json:"stopRebalanceUri"`
	NodeStatusesURI      string        `json:"nodeStatusesUri"`
	MaxBucketCount       int64         `json:"maxBucketCount"`
}

func eventMapping(content []byte) mapstr.M {
	var d Data
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: %+v", err)
	}

	logp.Info("Printing Data:")
	event := mapstr.M{
		"hdd": mapstr.M{
			"quota": mapstr.M{
				"total": mapstr.M{
					"bytes": d.StorageTotals.Hdd.QuotaTotal,
				},
			},
			"free": mapstr.M{
				"bytes": d.StorageTotals.Hdd.Free,
			},
			"total": mapstr.M{
				"bytes": d.StorageTotals.Hdd.Total,
			},
			"used": mapstr.M{
				"value": mapstr.M{
					"bytes": d.StorageTotals.Hdd.Used,
				},
				"by_data": mapstr.M{
					"bytes": d.StorageTotals.Hdd.UsedByData,
				},
			},
		},
		"max_bucket_count": d.MaxBucketCount,
		"quota": mapstr.M{
			"index_memory": mapstr.M{
				"mb": d.IndexMemoryQuota,
			},
			"memory": mapstr.M{
				"mb": d.MemoryQuota,
			},
		},
		"ram": mapstr.M{
			"quota": mapstr.M{
				"total": mapstr.M{
					"value": mapstr.M{
						"bytes": d.StorageTotals.RAM.QuotaTotal,
					},
					"per_node": mapstr.M{
						"bytes": d.StorageTotals.RAM.QuotaTotalPerNode,
					},
				},
				"used": mapstr.M{
					"value": mapstr.M{
						"bytes": d.StorageTotals.RAM.QuotaUsed,
					},
					"per_node": mapstr.M{
						"bytes": d.StorageTotals.RAM.QuotaUsedPerNode,
					},
				},
			},
			"total": mapstr.M{
				"bytes": d.StorageTotals.RAM.Total,
			},
			"used": mapstr.M{
				"value": mapstr.M{
					"bytes": d.StorageTotals.RAM.Used,
				},
				"by_data": mapstr.M{
					"bytes": d.StorageTotals.RAM.UsedByData,
				},
			},
		},
	}

	return event
}
