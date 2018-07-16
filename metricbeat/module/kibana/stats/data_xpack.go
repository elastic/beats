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

package stats

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/xpack"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schemaXPackMonitoring = s.Schema{
		"concurrent_connections": c.Int("concurrent_connections"),
		"os": c.Dict("os", s.Schema{
			"load": c.Dict("load", s.Schema{
				"1m":  c.Float("1m"),
				"5m":  c.Float("5m"),
				"15m": c.Float("15m"),
			}),
			"memory": c.Dict("memory", s.Schema{
				"total_in_bytes": c.Int("total_bytes"),
				"free_in_bytes":  c.Int("free_bytes"),
				"used_in_bytes":  c.Int("used_bytes"),
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
		}),
		"process": c.Dict("process", s.Schema{
			"event_loop_delay": c.Float("event_loop_delay"),
			"memory": c.Dict("memory", s.Schema{
				"heap": c.Dict("heap", s.Schema{
					"total_in_bytes": c.Int("total_bytes"),
					"used_in_bytes":  c.Int("used_bytes"),
					"size_limit":     c.Int("size_limit"),
				}),
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
		}),
		"requests": RequestsDict,
		"response_times": c.Dict("response_times", s.Schema{
			"average": c.Float("avg_ms"),
			"max":     c.Float("max_ms"),
		}, c.DictOptional),
		"sockets": SocketsDict,
		"kibana":  KibanaDict,
		"usage": c.Dict("usage", s.Schema{
			"index": c.Str("kibana.index"),
			"index_pattern": c.Dict("kibana.index_pattern", s.Schema{
				"total": c.Int("total"),
			}),
			"search": c.Dict("kibana.search", s.Schema{
				"total": c.Int("total"),
			}),
			"visualization": c.Dict("kibana.visualization", s.Schema{
				"total": c.Int("total"),
			}),
			"dashboard": c.Dict("kibana.dashboard", s.Schema{
				"total": c.Int("total"),
			}),
			"timelion_sheet": c.Dict("kibana.timelion_sheet", s.Schema{
				"total": c.Int("total"),
			}),
			"graph_workspace": c.Dict("kibana.graph_workspace", s.Schema{
				"total": c.Int("total"),
			}),
			"xpack": s.Object{
				"reporting": ReportingUsageDict,
			},
		}),
	}
)

func eventMappingXPack(r mb.ReporterV2, intervalMs int64, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	kibanaStatsFields, err := schemaXPackMonitoring.Apply(data)
	if err != nil {
		r.Error(err)
		return err
	}

	process := data["process"].(map[string]interface{})
	memory := process["memory"].(map[string]interface{})
	kibanaStatsFields.Put("process.memory.resident_set_size_in_bytes", int(memory["resident_set_size_bytes"].(float64)))

	timestamp := time.Now()
	kibanaStatsFields.Put("timestamp", timestamp)

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": data["cluster_uuid"].(string),
		"timestamp":    timestamp,
		"interval_ms":  intervalMs,
		"type":         "kibana_stats",
		"kibana_stats": kibanaStatsFields,
	}

	event.Index = xpack.MakeMonitoringIndexName(xpack.Kibana)
	r.Event(event)

	return nil
}
