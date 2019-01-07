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

package varz

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Varz will output server information on the monitoring port at /varz.
type Varz struct {
	ServerID         string         `json:"server_id"`
	Now              time.Time      `json:"now"`
	Uptime           string         `json:"uptime"`
	Mem              int            `json:"mem"`
	Cores            int            `json:"cores"`
	CPU              int            `json:"cpu"`
	TotalConnections int            `json:"total_connections"`
	Remotes          int            `json:"remotes"`
	InMsgsIn         int            `json:"in_msgs,omitempty"`
	InMsgs           int            `json:"msgs.in"`
	OutMsgsIn        int            `json:"out_msgs,omitempty"`
	OutMsgs          int            `json:"msgs.out"`
	InBytesIn        int            `json:"in_bytes,omitempty"`
	InBytes          int            `json:"bytes.in"`
	OutBytesIn       int            `json:"out_bytes,omitempty"`
	OutBytes         int            `json:"bytes.out"`
	SlowConsumers    int            `json:"slow_consumers"`
	HTTPReqStats     map[string]int `json:"http_req_stats,omitempty"`
	RootUriHits      int            `json:"http_req_stats.root_uri"`
	ConnzUriHits     int            `json:"http_req_stats.connz_uri"`
	RoutezUriHits    int            `json:"http_req_stats.routez_uri"`
	SubszUriHits     int            `json:"http_req_stats.subsz_uri"`
	VarzUriHits      int            `json:"http_req_stats.varz_uri"`
}

func eventMapping(content []byte) common.MapStr {
	var data Varz
	json.Unmarshal(content, &data)

	data.InMsgs = data.InMsgsIn
	data.InMsgsIn = 0
	data.OutMsgs = data.OutMsgsIn
	data.OutMsgsIn = 0
	data.InBytes = data.InBytesIn
	data.InBytesIn = 0
	data.OutBytes = data.OutBytesIn
	data.OutBytesIn = 0
	data.RootUriHits = data.HTTPReqStats["/"]
	data.ConnzUriHits = data.HTTPReqStats["/connz"]
	data.RoutezUriHits = data.HTTPReqStats["/routez"]
	data.SubszUriHits = data.HTTPReqStats["/subsz"]
	data.VarzUriHits = data.HTTPReqStats["/varz"]
	data.HTTPReqStats = make(map[string]int, 0)

	// TODO: add error handling
	event := common.MapStr{
		"metrics": data,
	}
	return event
}
