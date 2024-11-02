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
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/elastic-agent-libs/mapstr"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var (
	schema = s.Schema{
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
			"distro":          c.Str("distro", s.Optional),
			"distroRelease":   c.Str("distro_release", s.Optional),
			"platform":        c.Str("platform", s.Optional),
			"platformRelease": c.Str("platform_release", s.Optional),
			"cpuacct": c.Dict("cpuacct", s.Schema{
				"control_group": c.Str("control_group"),
				"usage_nanos":   c.Int("usage_nanos"),
			}, c.DictOptional),
			"cgroup_memory": c.Dict("cgroup_memory", s.Schema{
				"current_in_bytes":      c.Int("current_bytes"),
				"swap_current_in_bytes": c.Int("swap_current_bytes"),
			}, c.DictOptional),
		}),
		"kibana": c.Ifc("kibana"),
		"elasticsearch_client": c.Dict("elasticsearch_client", s.Schema{
			"total_active_sockets":  c.Int("total_active_sockets"),
			"total_idle_sockets":    c.Int("total_idle_sockets"),
			"total_queued_requests": c.Int("total_queued_requests"),
		}, c.DictOptional),
		"uuid":  c.Str("kibana.uuid"),
		"name":  c.Str("kibana.name"),
		"index": c.Str("kibana.index"),
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
			"event_loop_utilization": c.Dict("event_loop_utilization", s.Schema{
				"active":      c.Float("active"),
				"idle":        c.Float("idle"),
				"utilization": c.Float("utilization"),
			}, c.DictOptional),
			"memory": c.Dict("memory", s.Schema{
				"resident_set_size": s.Object{
					"bytes": c.Int("resident_set_size_bytes"),
				},
				"array_buffers": s.Object{
					"bytes": c.Int("array_buffers_bytes", s.Optional),
				},
				"external": s.Object{
					"bytes": c.Int("external_bytes", s.Optional),
				},
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

func eventMapping(r mb.ReporterV2, content []byte, isXpack bool) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return fmt.Errorf("failure parsing Kibana Stats API response: %w", err)
	}

	dataFields, err := schema.Apply(data)
	if err != nil {
		return fmt.Errorf("failure to apply stats schema: %w", err)
	}

	event := mb.Event{ModuleFields: mapstr.M{}, RootFields: mapstr.M{}}

	// Set elasticsearch cluster id
	elasticsearchClusterID, ok := data["cluster_uuid"]
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("cluster_uuid", elastic.Kibana)
		return event.Error
	}
	_, _ = event.ModuleFields.Put("elasticsearch.cluster.id", elasticsearchClusterID)

	// Set service ID
	uuid, err := dataFields.GetValue("uuid")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.uuid", elastic.Kibana)
		return event.Error
	}
	_, _ = event.RootFields.Put("service.id", uuid)
	_ = dataFields.Delete("uuid")

	// Set service version
	version, err := dataFields.GetValue("version")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.version", elastic.Kibana)
		return event.Error
	}
	_, _ = event.RootFields.Put("service.version", version)
	_ = dataFields.Delete("version")

	// Set service address
	serviceAddress, err := dataFields.GetValue("kibana.transport_address")
	if err != nil {
		event.Error = elastic.MakeErrorForMissingField("kibana.transport_address", elastic.Kibana)
		return event.Error
	}
	_, _ = event.RootFields.Put("service.address", serviceAddress)
	event.Host = fmt.Sprintf("%v", serviceAddress)

	// Set process PID
	process, ok := data["process"].(map[string]interface{})
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("process", elastic.Kibana)
		return event.Error
	}
	pid, ok := process["pid"].(float64)
	if !ok {
		event.Error = elastic.MakeErrorForMissingField("process.pid", elastic.Kibana)
		return event.Error
	}
	_, _ = event.RootFields.Put("process.pid", int(pid))

	_ = dataFields.Delete("kibana")

	event.MetricSetFields = dataFields

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Kibana)
		event.Index = index
	}

	r.Event(event)

	return nil
}
