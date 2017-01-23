package df

import (
	"encoding/json"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Stats struct {
	BytesUsed int64 `json:"bytes_used"`
	MaxAvail  int64 `json:"max_avail"`
	Objects   int64 `json:"objects"`
	KbUsed    int64 `json:"kb_used"`
}

type Pool struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Stats Stats  `json:"stats"`
}

type StatsCluster struct {
	TotalUsedBytes  int64 `json:"total_used_bytes"`
	TotalBytes      int64 `json:"total_bytes"`
	TotalAvailBytes int64 `json:"total_avail_bytes"`
}

type Output struct {
	StatsCluster StatsCluster `json:"stats"`
	Pools        []Pool       `json:"pools"`
}

type DfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(body io.Reader) []common.MapStr {

	var d DfRequest
	err := json.NewDecoder(body).Decode(&d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	event := common.MapStr{
		"stats": common.MapStr{
			"used": common.MapStr{
				"bytes": d.Output.StatsCluster.TotalUsedBytes,
			},
			"total": common.MapStr{
				"bytes": d.Output.StatsCluster.TotalBytes,
			},
			"available": common.MapStr{
				"bytes": d.Output.StatsCluster.TotalAvailBytes,
			},
		},
	}

	events = append(events, event)

	for _, Pool := range d.Output.Pools {
		event := common.MapStr{
			"pool": common.MapStr{
				"name": Pool.Name,
				"id":   Pool.Id,
				"stats": common.MapStr{
					"used": common.MapStr{
						"bytes": Pool.Stats.BytesUsed,
						"kb":    Pool.Stats.KbUsed,
					},
					"available": common.MapStr{
						"bytes": Pool.Stats.MaxAvail,
					},
					"objects": Pool.Stats.Objects,
				},
			},
		}

		events = append(events, event)

	}

	return events
}
