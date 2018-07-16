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
		"cluster_uuid": c.Str("cluster_uuid"),
		"name":         c.Str("name"),
		"uuid":         c.Str("uuid"),
		"version": c.Dict("version", s.Schema{
			"number": c.Str("number"),
		}),
		"status": c.Dict("status", s.Schema{
			"overall": c.Dict("overall", s.Schema{
				"state": c.Str("state"),
			}),
		}),
		"response_times": c.Dict("response_times", s.Schema{
			"avg": s.Object{
				"ms": c.Float("avg_in_millis"),
			},
			"max": s.Object{
				"ms": c.Int("max_in_millis"),
			},
		}),
		"requests": c.Dict("requests", s.Schema{
			"total":       c.Int("total", s.Optional),
			"disconnects": c.Int("disconnects", s.Optional),
		}),
		"concurrent_connections": c.Int("concurrent_connections"),
		"sockets": c.Dict("sockets", s.Schema{
			"http": c.Dict("http", s.Schema{
				"total": c.Int("total"),
			}),
			"https": c.Dict("https", s.Schema{
				"total": c.Int("total"),
			}),
		}),
		"event_loop_delay": c.Float("event_loop_delay"),
		"process": c.Dict("process", s.Schema{
			"memory": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"max": s.Object{
						"bytes": c.Int("heap_max_in_bytes"),
					},
					"used": s.Object{
						"bytes": c.Int("heap_used_in_bytes"),
					},
				},
				"resident_set_size": s.Object{
					"bytes": c.Int("resident_set_size_in_bytes"),
				},
				"external": s.Object{
					"bytes": c.Int("external_in_bytes"),
				},
			}),
			"pid": c.Int("pid"),
			"uptime": s.Object{
				"ms": c.Int("uptime_ms"),
			},
		}),
	}
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
