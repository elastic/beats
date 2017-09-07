package cluster_disk

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type StatsCluster struct {
	TotalUsedBytes  int64 `json:"total_used_bytes"`
	TotalBytes      int64 `json:"total_bytes"`
	TotalAvailBytes int64 `json:"total_avail_bytes"`
}

type Output struct {
	StatsCluster StatsCluster `json:"stats"`
}

type DfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventMapping(content []byte) common.MapStr {
	var d DfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	return common.MapStr{
		"used": common.MapStr{
			"bytes": d.Output.StatsCluster.TotalUsedBytes,
		},
		"total": common.MapStr{
			"bytes": d.Output.StatsCluster.TotalBytes,
		},
		"available": common.MapStr{
			"bytes": d.Output.StatsCluster.TotalAvailBytes,
		},
	}
}
