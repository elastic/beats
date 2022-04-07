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

package osd_df

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
)

// Node represents a node object
type Node struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Used        int64  `json:"kb_used"`
	Available   int64  `json:"kb_avail"`
	Total       int64  `json:"kb"`
	PgNum       int64  `json:"pgs"`
	DeviceClass string `json:"device_class"`
}

// Output contains a node list from the df response
type Output struct {
	Nodes []Node `json:"nodes"`
}

// OsdDfRequest contains the df response
type OsdDfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var d OsdDfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		return nil, errors.Wrap(err, "error getting data for OSD_DF")
	}

	nodeList := d.Output.Nodes

	//osd node list
	events := []common.MapStr{}
	for _, node := range nodeList {
		nodeInfo := common.MapStr{
			"id":             node.ID,
			"name":           node.Name,
			"total.byte":     node.Total,
			"used.byte":      node.Used,
			"available.byte": node.Available,
			"device_class":   node.DeviceClass,
			"pg_num":         node.PgNum,
		}

		if 0 != node.Total {
			var usedPct float64
			usedPct = float64(node.Used) / float64(node.Total)
			nodeInfo["used.pct"] = usedPct
		}

		events = append(events, nodeInfo)
	}

	return events, nil
}
