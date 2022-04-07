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

package collstats

import (
	"errors"
	"strings"

	"github.com/elastic/beats/v8/libbeat/common"
)

func eventMapping(key string, data common.MapStr) (common.MapStr, error) {
	names := strings.SplitN(key, ".", 2)

	if len(names) < 2 {
		return nil, errors.New("Collection name invalid")
	}

	event := common.MapStr{
		"db":         names[0],
		"collection": names[1],
		"name":       key,
		"total": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "total.time"),
			},
			"count": mustGetMapStrValue(data, "total.count"),
		},
		"lock": common.MapStr{
			"read": common.MapStr{
				"time": common.MapStr{
					"us": mustGetMapStrValue(data, "readLock.time"),
				},
				"count": mustGetMapStrValue(data, "readLock.count"),
			},
			"write": common.MapStr{
				"time": common.MapStr{
					"us": mustGetMapStrValue(data, "writeLock.time"),
				},
				"count": mustGetMapStrValue(data, "writeLock.count"),
			},
		},
		"queries": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "queries.time"),
			},
			"count": mustGetMapStrValue(data, "queries.count"),
		},
		"getmore": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "getmore.time"),
			},
			"count": mustGetMapStrValue(data, "getmore.count"),
		},
		"insert": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "insert.time"),
			},
			"count": mustGetMapStrValue(data, "insert.count"),
		},
		"update": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "update.time"),
			},
			"count": mustGetMapStrValue(data, "update.count"),
		},
		"remove": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "remove.time"),
			},
			"count": mustGetMapStrValue(data, "remove.count"),
		},
		"commands": common.MapStr{
			"time": common.MapStr{
				"us": mustGetMapStrValue(data, "commands.time"),
			},
			"count": mustGetMapStrValue(data, "commands.count"),
		},
	}

	return event, nil
}

func mustGetMapStrValue(m common.MapStr, key string) interface{} {
	v, _ := m.GetValue(key)
	return v
}
