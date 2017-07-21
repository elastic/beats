package pool_disk

import (
	"encoding/json"

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

type Output struct {
	Pools []Pool `json:"pools"`
}

type DfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) []common.MapStr {
	var d DfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	events := []common.MapStr{}

	for _, Pool := range d.Output.Pools {
		event := common.MapStr{
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
		}

		events = append(events, event)

	}

	return events
}
