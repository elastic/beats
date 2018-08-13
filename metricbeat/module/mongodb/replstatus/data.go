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
	"github.com/elastic/beats/libbeat/common"
)

func eventMapping(oplogInfo oplogInfo, replStatus MongoReplStatus) common.MapStr {
	var result common.MapStr = make(common.MapStr)

	result["oplog"] = common.MapStr{
		"size": common.MapStr{
			"allocated": oplogInfo.allocated,
			"used":      oplogInfo.used,
		},
		"first": common.MapStr{
			"timestamp": oplogInfo.firstTs,
		},
		"last": common.MapStr{
			"timestamp": oplogInfo.lastTs,
		},
		"window": oplogInfo.diff,
	}
	result["set_name"] = replStatus.Set
	result["server_date"] = replStatus.Date
	result["optimes"] = common.MapStr{
		"last_committed": replStatus.OpTimes.LastCommitted.getTimeStamp(),
		"applied":        replStatus.OpTimes.Applied.getTimeStamp(),
		"durable":        replStatus.OpTimes.Durable.getTimeStamp(),
	}

	// find lag and headroom
	minLag, maxLag, lagIsOk := findLag(replStatus.Members)
	if lagIsOk {
		result["lag"] = common.MapStr{
			"max": maxLag,
			"min": minLag,
		}

		result["headroom"] = common.MapStr{
			"max": oplogInfo.diff - minLag,
			"min": oplogInfo.diff - maxLag,
		}
	} else {
		result["lag"] = common.MapStr{
			"max": nil,
			"min": nil,
		}

		result["headroom"] = common.MapStr{
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

	result["members"] = common.MapStr{
		"primary": common.MapStr{
			"host":   findHostsByState(replStatus.Members, PRIMARY)[0],
			"optime": findOptimesByState(replStatus.Members, PRIMARY)[0],
		},
		"secondary": common.MapStr{
			"hosts":   secondaryHosts,
			"count":   len(secondaryHosts),
			"optimes": findOptimesByState(replStatus.Members, SECONDARY),
		},
		"recovering": common.MapStr{
			"hosts": recoveringHosts,
			"count": len(recoveringHosts),
		},
		"unknown": common.MapStr{
			"hosts": unknownHosts,
			"count": len(unknownHosts),
		},
		"startup2": common.MapStr{
			"hosts": startup2Hosts,
			"count": len(startup2Hosts),
		},
		"arbiter": common.MapStr{
			"hosts": arbiterHosts,
			"count": len(arbiterHosts),
		},
		"down": common.MapStr{
			"hosts": downHosts,
			"count": len(downHosts),
		},
		"rollback": common.MapStr{
			"hosts": rollbackHosts,
			"count": len(rollbackHosts),
		},
		"unhealthy": common.MapStr{
			"hosts": unhealthyHosts,
			"count": len(unhealthyHosts),
		},
	}

	return result
}
