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

package keyspace

import (
	"strings"

	"github.com/menderesk/beats/v7/libbeat/common"
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstrstr"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/redis"
)

// Map data to MapStr
func eventsMapping(r mb.ReporterV2, info map[string]string) {
	for key, space := range getKeyspaceStats(info) {
		space["id"] = key
		event := mb.Event{
			MetricSetFields: space,
		}
		r.Event(event)
	}
}

func getKeyspaceStats(info map[string]string) map[string]common.MapStr {
	keyspaceMap := findKeyspaceStats(info)
	return parseKeyspaceStats(keyspaceMap)
}

// findKeyspaceStats will grep for keyspace ("^db" keys) and return the resulting map
func findKeyspaceStats(info map[string]string) map[string]string {
	keyspace := map[string]string{}

	for k, v := range info {
		if strings.HasPrefix(k, "db") {
			keyspace[k] = v
		}
	}
	return keyspace
}

var schema = s.Schema{
	"keys":    c.Int("keys"),
	"expires": c.Int("expires"),
	"avg_ttl": c.Int("avg_ttl"),
}

// parseKeyspaceStats resolves the overloaded value string that Redis returns for keyspace
func parseKeyspaceStats(keyspaceMap map[string]string) map[string]common.MapStr {
	keyspace := map[string]common.MapStr{}
	for k, v := range keyspaceMap {

		// Extract out the overloaded values for db keyspace
		// fmt: info[db0] = keys=795341,expires=0,avg_ttl=0
		dbInfo := redis.ParseRedisLine(v, ",")

		if len(dbInfo) == 3 {
			db := map[string]interface{}{}
			for _, dbEntry := range dbInfo {
				stats := redis.ParseRedisLine(dbEntry, "=")

				if len(stats) == 2 {
					db[stats[0]] = stats[1]
				}
			}
			data, _ := schema.Apply(db)
			keyspace[k] = data
		}
	}
	return keyspace
}
