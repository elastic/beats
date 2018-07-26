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

package slowlog

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	rd "github.com/garyburd/redigo/redis"
)

// log contains all data related to one slowlog entry
type log struct {
	id            int64
	timestamp     int64
	duration      int
	cmd           string
	key           string
	args          []string
	clientAddress string
	clientName    string
}

// Map data to MapStr
func eventMapping(slowlogs []interface{}) []common.MapStr {
	var events []common.MapStr
	for _, item := range slowlogs {
		entry, err := rd.Values(item, nil)
		if err != nil {
			logp.Err("Error loading slowlog values: %s", err)
			continue
		}

		var log log
		var args []string
		rd.Scan(entry, &log.id, &log.timestamp, &log.duration, &args, &log.clientAddress, &log.clientName)

		// This splits up the args into cmd, key, args.
		argsLen := len(args)
		if argsLen > 0 {
			log.cmd = args[0]
		}
		if argsLen > 1 {
			log.key = args[1]
		}

		// This could contain confidential data, processors should be used to drop it if needed
		if argsLen > 2 {
			log.args = args[2:]
		}

		event := common.MapStr{
			"id":      log.id,
			"command": log.cmd,
			"key":     log.key,
			"duration": common.MapStr{
				"us": log.duration,
			},
			"timestamp": log.timestamp,
			"client": common.MapStr{
				"name":    log.clientName,
				"address": log.clientAddress,
			},
		}

		if log.args != nil {
			event["args"] = log.args
		}

		events = append(events, event)
	}

	return events
}
