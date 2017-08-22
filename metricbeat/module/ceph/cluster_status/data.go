package cluster_status

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type PgState struct {
	Count     int64  `json:"count"`
	StateName string `json:"state_name"`
}

type Pgmap struct {
	AvailByte int64 `json:"bytes_avail"`
	TotalByte int64 `json:"bytes_total"`
	UsedByte  int64 `json:"bytes_used"`
	DataByte  int64 `json:"data_bytes"`

	DegradedObjs  int64   `json:"degraded_objects"`
	DegradedRatio float64 `json:"degraded_ratio"`
	DegradedTotal int64   `json:"degraded_total"`

	MisplacedObjs  int64   `json:"misplaced_objects"`
	MisplacedRatio float64 `json:"misplaced_ratio"`
	MisplacedTotal int64   `json:"misplaced_total"`

	ReadByteSec  int64 `json:"read_bytes_sec"`
	ReadOpSec    int64 `json:"read_op_per_sec"`
	WriteByteSec int64 `json:"write_bytes_sec"`
	WriteOpSec   int64 `json:"write_op_per_sec"`
	Version      int64 `json:"version"`

	PgNum    int64     `json:"num_pgs"`
	PgStates []PgState `json:"pgs_by_state"`
}

type Output struct {
	Pgmap Pgmap `json:"pgmap"`
}

type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventMapping(content []byte) []common.MapStr {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
	}

	pgmap := d.Output.Pgmap

	traffic := common.MapStr{}
	traffic["read_bytes_sec"] = pgmap.ReadByteSec
	traffic["read_op_per_sec"] = pgmap.ReadOpSec
	traffic["write_bytes_sec"] = pgmap.WriteByteSec
	traffic["write_op_per_sec"] = pgmap.WriteOpSec

	misplace := common.MapStr{}
	misplace["objects"] = pgmap.MisplacedObjs
	misplace["ratio"] = pgmap.MisplacedRatio
	misplace["total"] = pgmap.MisplacedTotal

	degraded := common.MapStr{}
	degraded["objects"] = pgmap.DegradedObjs
	degraded["ratio"] = pgmap.DegradedRatio
	degraded["total"] = pgmap.DegradedTotal

	pg := common.MapStr{}
	pg["bytes_avail"] = pgmap.AvailByte
	pg["bytes_total"] = pgmap.TotalByte
	pg["bytes_used"] = pgmap.UsedByte
	pg["data_bytes"] = pgmap.DataByte

	pg_event := common.MapStr{}
	pg_event["traffic"] = traffic
	pg_event["misplace"] = misplace
	pg_event["degraded"] = degraded
	pg_event["pg"] = pg
	pg_event["version"] = pgmap.Version

	events := []common.MapStr{}
	events = append(events, pg_event)

	for _, state := range pgmap.PgStates {
		state_evn := common.MapStr{
			"count":      state.Count,
			"state_name": state.StateName,
		}
		events = append(events, state_evn)
	}

	return events
}
