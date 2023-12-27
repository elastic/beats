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

package self

import (
	"encoding/json"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type LeaderInfo struct {
	Leader    string `json:"leader"`
	StartTime string `json:"startTime"`
	Uptime    string `json:"uptime"`
}

type AppendRequest struct {
	Count int64 `json:"recvAppendRequestCnt"`
}

type Recv struct {
	Appendrequest AppendRequest
	Bandwidthrate float64 `json:"recvBandwidthRate"`
	Pkgrate       float64 `json:"recvPkgRate"`
}

type sendAppendRequest struct {
	Cnt int64 `json:"sendAppendRequestCnt"`
}

type Send struct {
	AppendRequest sendAppendRequest
	BandwidthRate float64 `json:"sendBandwidthRate"`
	PkgRate       float64 `json:"sendPkgRate"`
}

type Self struct {
	ID         string `json:"id"`
	LeaderInfo LeaderInfo
	Name       string `json:"name"`
	Recv       Recv
	Send       Send
	StartTime  string `json:"startTime"`
	State      string `json:"state"`
}

func eventMapping(content []byte) mapstr.M {
	var data Self
	json.Unmarshal(content, &data)
	event := mapstr.M{
		"id": data.ID,
		"leaderinfo": mapstr.M{
			"leader":    data.LeaderInfo.Leader,
			"starttime": data.LeaderInfo.StartTime,
			"uptime":    data.LeaderInfo.Uptime,
		},
		"name": data.Name,
		"recv": mapstr.M{
			"appendrequest": mapstr.M{
				"count": data.Recv.Appendrequest.Count,
			},
			"bandwidthrate": data.Recv.Bandwidthrate,
			"pkgrate":       data.Recv.Pkgrate,
		},
		"send": mapstr.M{
			"appendrequest": mapstr.M{
				"count": data.Send.AppendRequest.Cnt,
			},
			"bandwidthrate": data.Send.BandwidthRate,
			"pkgrate":       data.Send.PkgRate,
		},
		"starttime": data.StartTime,
		"state":     data.State,
	}

	return event
}
