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

package cluster_status

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// PgState represents placement group state
type PgState struct {
	Count     int64  `json:"count"`
	StateName string `json:"state_name"`
}

// Pgmap represents data from a placement group
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

// Osdmap represents data from an OSD
type Osdmap struct {
	Epoch      int64 `json:"epoch"`
	Full       bool  `json:"full"`
	Nearfull   bool  `json:"nearfull"`
	OsdNum     int64 `json:"num_osds"`
	UpOsds     int64 `json:"num_up_osds"`
	InOsds     int64 `json:"num_in_osds"`
	RemapedPgs int64 `json:"num_remapped_pgs"`
}

// Osdmap_ is a placeholder for the json parser
type Osdmap_ struct {
	Osdmap Osdmap `json:"osdmap"`
}

// Output is the response body
type Output struct {
	Pgmap  Pgmap   `json:"pgmap"`
	Osdmap Osdmap_ `json:"osdmap"`
}

// HealthRequest represents the response to a health request
type HealthRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var d HealthRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		return nil, errors.Wrap(err, "error getting HealthRequest data")
	}

	//osd map info
	osdmap := d.Output.Osdmap.Osdmap

	osdState := common.MapStr{}
	osdState["epoch"] = osdmap.Epoch
	osdState["full"] = osdmap.Full
	osdState["nearfull"] = osdmap.Nearfull
	osdState["osd_count"] = osdmap.OsdNum
	osdState["up_osd_count"] = osdmap.UpOsds
	osdState["in_osd_count"] = osdmap.InOsds
	osdState["remapped_pg_count"] = osdmap.RemapedPgs

	//pg map info
	pgmap := d.Output.Pgmap

	traffic := common.MapStr{}
	traffic["read_bytes"] = pgmap.ReadByteSec
	traffic["read_op_per_sec"] = pgmap.ReadOpSec
	traffic["write_bytes"] = pgmap.WriteByteSec
	traffic["write_op_per_sec"] = pgmap.WriteOpSec

	misplace := common.MapStr{}
	misplace["objects"] = pgmap.MisplacedObjs
	misplace["pct"] = pgmap.MisplacedRatio
	misplace["total"] = pgmap.MisplacedTotal

	degraded := common.MapStr{}
	degraded["objects"] = pgmap.DegradedObjs
	degraded["pct"] = pgmap.DegradedRatio
	degraded["total"] = pgmap.DegradedTotal

	pg := common.MapStr{}
	pg["avail_bytes"] = pgmap.AvailByte
	pg["total_bytes"] = pgmap.TotalByte
	pg["used_bytes"] = pgmap.UsedByte
	pg["data_bytes"] = pgmap.DataByte

	stateEvent := common.MapStr{}
	stateEvent["osd"] = osdState
	stateEvent["traffic"] = traffic
	stateEvent["misplace"] = misplace
	stateEvent["degraded"] = degraded
	stateEvent["pg"] = pg
	stateEvent["version"] = pgmap.Version

	events := []common.MapStr{}
	events = append(events, stateEvent)

	//pg state info
	for _, state := range pgmap.PgStates {
		stateEvn := common.MapStr{
			"count":      state.Count,
			"state_name": state.StateName,
			"version":    pgmap.Version,
		}
		evt := common.MapStr{
			"pg_state": stateEvn,
		}
		events = append(events, evt)
	}

	return events, nil
}
