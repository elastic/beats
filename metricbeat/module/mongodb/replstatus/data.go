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

// var schema = s.Schema{
// 	"oplog": c.Dict("oplog", s.Schema{
// 		"size": s.Object{
// 			"allocated": c.Int("logSize"),
// 			"used":      c.Int("used"),
// 		},
// 		"first": s.Object{
// 			"timestamp": c.Int("tFirst"),
// 		},
// 		"last": s.Object{
// 			"timestamp": c.Int("tLast"),
// 		},
// 		"window": c.Int("timeDiff"),
// 	}),
// 	"set_name":    c.Str("setName"),
// 	"server_date": c.Int("serverDate"),
// 	"operation_times": c.Dict("operationTimes", s.Schema{
// 		"last_committed": c.Int("lastCommitted"),
// 		"applied":        c.Int("applied"),
// 		"durable":        c.Int("durable"),
// 	}),
// 	"states": c.Dict("stateCount", s.Schema{
// 		"secondary": s.Object{
// 			"count": c.Int("secondary"),
// 		},
// 	}),
// 	// ToDo add []string
// 	// "unhealthy": s.Object {
// 	// 	"hosts": c.Str()
// 	// }
// }

func eventMapping(oplog oplog, replStatus ReplStatusRaw) common.MapStr {
	var result common.MapStr = make(common.MapStr)

	result["oplog"] = map[string]interface{}{
		"logSize":  oplog.allocated,
		"used":     oplog.used,
		"tFirst":   oplog.firstTs,
		"tLast":    oplog.lastTs,
		"timeDiff": oplog.diff,
	}
	result["set_name"] = replStatus.Set
	result["server_date"] = replStatus.Date.Unix()
	result["optimes"] = map[string]interface{}{
		// ToDo find actual timestamps
		"lastCommitted": replStatus.OpTimes.LastCommitted.Ts,
		"applied":       replStatus.OpTimes.Applied.Ts,
		"durable":       replStatus.OpTimes.Durable.Ts,
	}
	result["members"] = map[string]interface{}{
		"secondary": map[string]interface{}{"count": len(findHostsByState(replStatus.Members, SECONDARY))},
		"unhealthy": findUnhealthyHosts(replStatus.Members),
	}

	return result
}
