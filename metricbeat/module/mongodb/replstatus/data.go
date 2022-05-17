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

package replstatus

import (
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func eventMapping(oplogInfo oplogInfo, replStatus MongoReplStatus) mapstr.M {
	var result = make(mapstr.M)

	result["oplog"] = mapstr.M{
		"size": mapstr.M{
			"allocated": oplogInfo.allocated,
			"used":      oplogInfo.used,
		},
		"first": mapstr.M{
			"timestamp": oplogInfo.firstTs,
		},
		"last": mapstr.M{
			"timestamp": oplogInfo.lastTs,
		},
		"window": oplogInfo.diff,
	}
	result["set_name"] = replStatus.Set
	result["server_date"] = replStatus.Date
	result["optimes"] = mapstr.M{
		"last_committed": replStatus.OpTimes.LastCommitted.Ts.T,
		"applied":        replStatus.OpTimes.Applied.Ts.T,
		"durable":        replStatus.OpTimes.Durable.Ts.T,
	}

	// find lag and headroom
	minLag, maxLag, lagIsOk := findLag(replStatus.Members)
	if lagIsOk {
		result["lag"] = mapstr.M{
			"max": maxLag,
			"min": minLag,
		}

		result["headroom"] = mapstr.M{
			"max": int64(oplogInfo.diff) - minLag,
			"min": int64(oplogInfo.diff) - maxLag,
		}
	} else {
		result["lag"] = mapstr.M{
			"max": nil,
			"min": nil,
		}

		result["headroom"] = mapstr.M{
			"max": nil,
			"min": nil,
		}
	}

	var (
		secondaryHosts  = findHostsByState(replStatus.Members, SECONDARY)
		recoveringHosts = findHostsByState(replStatus.Members, RECOVERING)
		unknownHosts    = findHostsByState(replStatus.Members, UNKNOWN)
		startup2Hosts   = findHostsByState(replStatus.Members, STARTUP2)
		arbiterHosts    = findHostsByState(replStatus.Members, ARBITER)
		downHosts       = findHostsByState(replStatus.Members, DOWN)
		rollbackHosts   = findHostsByState(replStatus.Members, ROLLBACK)
		unhealthyHosts  = findUnhealthyHosts(replStatus.Members)
	)

	result["members"] = mapstr.M{
		"primary": mapstr.M{
			"host":   findHostsByState(replStatus.Members, PRIMARY)[0],
			"optime": findOptimesByState(replStatus.Members, PRIMARY)[0],
		},
		"secondary": mapstr.M{
			"hosts":   secondaryHosts,
			"count":   len(secondaryHosts),
			"optimes": findOptimesByState(replStatus.Members, SECONDARY),
		},
		"recovering": mapstr.M{
			"hosts": recoveringHosts,
			"count": len(recoveringHosts),
		},
		"unknown": mapstr.M{
			"hosts": unknownHosts,
			"count": len(unknownHosts),
		},
		"startup2": mapstr.M{
			"hosts": startup2Hosts,
			"count": len(startup2Hosts),
		},
		"arbiter": mapstr.M{
			"hosts": arbiterHosts,
			"count": len(arbiterHosts),
		},
		"down": mapstr.M{
			"hosts": downHosts,
			"count": len(downHosts),
		},
		"rollback": mapstr.M{
			"hosts": rollbackHosts,
			"count": len(rollbackHosts),
		},
		"unhealthy": mapstr.M{
			"hosts": unhealthyHosts,
			"count": len(unhealthyHosts),
		},
	}

	return result
}
