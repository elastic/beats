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

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"cluster_uuid":           c.Str("cluster_uuid"),
		"concurrent_connections": c.Int("concurrent_connections"),
		"process": c.Dict("process", s.Schema{
			"event_loop_delay": s.Object{
				"ms": c.Float("event_loop_delay"),
			},
			"memory": c.Dict("memory", s.Schema{
				"heap": c.Dict("heap", s.Schema{
					"total": s.Object{
						"bytes": c.Int("total_bytes"),
					},
					"used": s.Object{
						"bytes": c.Int("used_bytes"),
					},
					"size_limit": s.Object{
						"bytes": c.Int("size_limit"),
					},
				}),
			}),
			"uptime": s.Object{
				"ms": c.Int("uptime_ms"),
			},
		}),
		"request": RequestsDict,
		"response_time": c.Dict("response_times", s.Schema{
			"avg": s.Object{
				"ms": c.Int("avg_ms", s.Optional),
			},
			"max": s.Object{
				"ms": c.Int("max_ms"),
			},
		}),
		"kibana": KibanaDict,
	}

	// RequestsDict defines how to convert the requests field
	RequestsDict = c.Dict("requests", s.Schema{
		"disconnects": c.Int("disconnects", s.Optional),
		"total":       c.Int("total", s.Optional),
	})

	// KibanaDict defines how to convert the kibana field
	KibanaDict = c.Dict("kibana", s.Schema{
		"uuid":              c.Str("uuid"),
		"name":              c.Str("name"),
		"index":             c.Str("index"),
		"host":              c.Str("host"),
		"transport_address": c.Str("transport_address"),
		"version":           c.Str("version"),
		"snapshot":          c.Bool("snapshot"),
		"status":            c.Str("status"),
	})
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		r.Error(err)
	}

	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", "kibana")

	// Set elasticsearch cluster id
	if clusterID, ok := dataFields["cluster_uuid"]; ok {
		delete(dataFields, "cluster_uuid")
		event.RootFields.Put("elasticsearch.cluster.id", clusterID)
	}

	event.MetricSetFields = dataFields

	r.Event(event)

	return err
}
