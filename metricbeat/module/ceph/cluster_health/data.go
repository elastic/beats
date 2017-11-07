package cluster_health

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Timecheck struct {
	RoundStatus string `json:"round_status"`
	Epoch       int64  `json:"epoch"`
	Round       int64  `json:"round"`
}

type Output struct {
	OverallStatus string    `json:"overall_status"`
	Timechecks    Timecheck `json:"timechecks"`
}

type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventMapping(content []byte) common.MapStr {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	return common.MapStr{
		"overall_status": d.Output.OverallStatus,
		"timechecks": common.MapStr{
			"epoch": d.Output.Timechecks.Epoch,
			"round": common.MapStr{
				"value":  d.Output.Timechecks.Round,
				"status": d.Output.Timechecks.RoundStatus,
			},
		},
	}
}
