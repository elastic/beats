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

package osd_tree

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Node represents a node object
type Node struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	TypeID   int64   `json:"type_id"`
	Children []int64 `json:"children"`

	CrushWeight     float64 `json:"crush_weight"`
	Depth           int64   `json:"depth"`
	Exist           int64   `json:"exists"`
	PrimaryAffinity float64 `json:"primary_affinity"`
	Reweight        float64 `json:"reweight"`
	Status          string  `json:"status"`
	DeviceClass     string  `json:"device_class"`
}

// Output contains a node list from the df response
type Output struct {
	Nodes []Node `json:"nodes"`
}

// OsdTreeRequest is a OSD response object
type OsdTreeRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var d OsdTreeRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: %+v", err)
		return nil, err
	}

	nodeList := d.Output.Nodes

	//generate fatherNode and children map
	fatherMap := make(map[string]string)
	childrenMap := make(map[string]string)

	for _, node := range nodeList {
		if node.ID >= 0 {
			//it's osd node
			continue
		}
		childrenList := []string{}
		for _, child := range node.Children {
			childIDStr := strconv.FormatInt(child, 10)
			childrenList = append(childrenList, childIDStr)
			fatherMap[childIDStr] = node.Name
		}
		//generate bucket node's children list
		childrenMap[node.Name] = strings.Join(childrenList, ",")
	}

	//osd node list
	events := []common.MapStr{}
	for _, node := range nodeList {
		nodeInfo := common.MapStr{}
		if node.ID < 0 {
			//bucket node
			nodeInfo["children"] = strings.Split(childrenMap[node.Name], ",")
		} else {
			//osd node
			nodeInfo["crush_weight"] = node.CrushWeight
			nodeInfo["depth"] = node.Depth
			nodeInfo["primary_affinity"] = node.PrimaryAffinity
			nodeInfo["reweight"] = node.Reweight
			nodeInfo["status"] = node.Status
			nodeInfo["device_class"] = node.DeviceClass
			if node.Exist > 0 {
				nodeInfo["exists"] = true
			} else {
				nodeInfo["exists"] = false
			}
		}
		nodeInfo["id"] = node.ID
		nodeInfo["name"] = node.Name
		nodeInfo["type"] = node.Type
		nodeInfo["type_id"] = node.TypeID

		idStr := strconv.FormatInt(node.ID, 10)
		nodeInfo["father"] = fatherMap[idStr]

		events = append(events, nodeInfo)
	}

	return events, nil
}
