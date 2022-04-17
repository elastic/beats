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

package leader

import (
	"encoding/json"

	"github.com/menderesk/beats/v7/libbeat/common"
)

type Counts struct {
	Success int64 `json:"success"`
	Fail    int64 `json:"fail"`
}

type Latency struct {
	Average           float64 `json:"average"`
	Current           float64 `json:"current"`
	Maximum           float64 `json:"maximum"`
	Minimum           int64   `json:"minimum"`
	StandardDeviation float64 `json:"standardDeviation"`
}

type FollowersID struct {
	Latency Latency `json:"latency"`
	Counts  Counts  `json:"counts"`
}

type Leader struct {
	Followers map[string]FollowersID `json:"followers"`
	Leader    string                 `json:"leader"`
}

func eventMapping(content []byte) common.MapStr {
	var data Leader
	json.Unmarshal(content, &data)
	event := common.MapStr{
		"followers": data.Followers,
		"leader":    data.Leader,
	}
	return event
}
