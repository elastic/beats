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

package mgr_osd_tree

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/metricbeat/module/ceph/mgr"
)

type OsdTreeResponse struct {
	Nodes []struct {
		ID              int64   `json:"id"`
		Name            string  `json:"name"`
		Type            string  `json:"type"`
		TypeID          int64   `json:"type_id"`
		Children        []int64 `json:"children"`
		CrushWeight     float64 `json:"crush_weight"`
		Depth           int64   `json:"depth"`
		Exist           int64   `json:"exists"`
		PrimaryAffinity float64 `json:"primary_affinity"`
		Reweight        float64 `json:"reweight"`
		Status          string  `json:"status"`
		DeviceClass     string  `json:"device_class"`
	} `json:"nodes"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var response OsdTreeResponse
	err := mgr.UnmarshalResponse(content, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not get response data")
	}

	nodeList := response.Nodes

	//generate fatherNode and children map
	fatherMap := make(map[string]string)
	childrenMap := make(map[string]string)

	for _, node := range nodeList {
		if node.ID >= 0 {
			continue // it's OSD node
		}
		var childrenList []string
		for _, child := range node.Children {
			childIDStr := strconv.FormatInt(child, 10)
			childrenList = append(childrenList, childIDStr)
			fatherMap[childIDStr] = node.Name
		}
		// generate bucket node's children list
		childrenMap[node.Name] = strings.Join(childrenList, ",")
	}

	// OSD node list
	var events []common.MapStr
	for _, node := range nodeList {
		nodeInfo := common.MapStr{}
		if node.ID < 0 {
			// bucket node
			nodeInfo["children"] = strings.Split(childrenMap[node.Name], ",")
		} else {
			// OSD node
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
