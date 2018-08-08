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

func eventMapping(oplog oplog, replStatus ReplStatusRaw) common.MapStr {
	var result common.MapStr = make(common.MapStr)

	result["oplog"] = map[string]interface{}{
		"size":  map[string]interface{} {
			"allocated": oplog.allocated,
			"used":     oplog.used,
		},
		"first": map[string]interface{} {
			"timestamp": oplog.firstTs,
		},
		"last":  map[string]interface{} {
			"timestamp": oplog.lastTs,
		},
		"window": oplog.diff,
	}
	result["set_name"] = replStatus.Set
	result["server_date"] = replStatus.Date
	result["optimes"] = map[string]interface{}{
		// ToDo find actual timestamps
		"last_committed": replStatus.OpTimes.LastCommitted.getTimeStamp(),
		"applied":       replStatus.OpTimes.Applied.getTimeStamp(),
		"durable":       replStatus.OpTimes.Durable.getTimeStamp(),
	}
	result["lag"] = map[string]interface{} {
		"max": findMaxLag(replStatus.Members),
		"min": findMinLag(replStatus.Members),
	}
	result["headroom"] = map[string]interface{} {
		"max": oplog.diff - findMinLag(replStatus.Members),
		"min": oplog.diff - findMaxLag(replStatus.Members),
	}

	var (
		secondaryHosts = findHostsByState(replStatus.Members, SECONDARY)
	 	recoveringHosts = findHostsByState(replStatus.Members, RECOVERING)
	 	unknownHosts = findHostsByState(replStatus.Members, UNKNOWN)
	 	startup2Hosts = findHostsByState(replStatus.Members, STARTUP2)
	 	arbiterHosts = findHostsByState(replStatus.Members, ARBITER)
	 	downHosts = findHostsByState(replStatus.Members, DOWN)
	 	rollbackHosts = findHostsByState(replStatus.Members, ROLLBACK)
		unhealthyHosts = findUnhealthyHosts(replStatus.Members)
	)

	result["members"] = map[string]interface{}{
		"primary": findHostsByState(replStatus.Members, PRIMARY)[0],
		"secondary": map[string]interface{}{
			"hosts": secondaryHosts,
			"count": len(secondaryHosts),
		},
		"recovering": map[string]interface{}{
			"hosts": recoveringHosts,
			"count": len(recoveringHosts),
		},
		"unknown": map[string]interface{}{
			"hosts": unknownHosts,
			"count": len(unknownHosts),
		},
		"startup2": map[string]interface{}{
			"hosts": startup2Hosts,
			"count": len(startup2Hosts),
		},
		"arbiter": map[string]interface{}{
			"hosts": arbiterHosts,
			"count": len(arbiterHosts),
		},
		"down": map[string]interface{}{
			"hosts": downHosts,
			"count": len(downHosts),
		},
		"rollback": map[string]interface{}{
			"hosts": rollbackHosts,
			"count": len(rollbackHosts),
		},
		"unhealthy": map[string]interface{} {
			"hosts": unhealthyHosts,
			"count": len(unhealthyHosts),
		},
	}

	return result
}
