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

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kibana"
)

var (
	schema = s.Schema{
		"uuid":  c.Str("kibana.uuid"),
		"name":  c.Str("kibana.name"),
		"index": c.Str("kibana.name"),
		"host": s.Object{
			"name": c.Str("kibana.host"),
		},
		"transport_address":      c.Str("kibana.transport_address"),
		"version":                c.Str("kibana.version"),
		"snapshot":               c.Bool("kibana.snapshot"),
		"status":                 c.Str("kibana.status"),
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
				"ms": c.Int("max_ms", s.Optional),
			},
		}),
	}

	// RequestsDict defines how to convert the requests field
	RequestsDict = c.Dict("requests", s.Schema{
		"disconnects": c.Int("disconnects", s.Optional),
		"total":       c.Int("total", s.Optional),
	})
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Kibana Stats API response")
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure to apply stats schema")
	}

	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", kibana.ModuleName)

	// Set elasticsearch cluster id
	elasticsearchClusterID, ok := data["cluster_uuid"]
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	event.RootFields.Put("elasticsearch.cluster.id", elasticsearchClusterID)

	// Set process PID
	process, ok := data["process"].(map[string]interface{})
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("process", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	pid, ok := process["pid"].(float64)
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("process.pid", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	event.RootFields.Put("process.pid", int(pid))

	// Set service ID
	uuid, err := dataFields.GetValue("uuid")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	event.RootFields.Put("service.id", uuid)
	dataFields.Delete("uuid")

	// Set service version
	version, err := dataFields.GetValue("version")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	event.RootFields.Put("service.version", version)
	dataFields.Delete("version")

	// Set service address
	serviceAddress, err := dataFields.GetValue("transport_address")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
		r.Event(event)
		return event.Error
	}
	event.RootFields.Put("service.address", serviceAddress)
	dataFields.Delete("transport_address")

	event.MetricSetFields = dataFields

	r.Event(event)
	return nil
}
