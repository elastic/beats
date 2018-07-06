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
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schemaXPack = s.Schema{
		"concurrent_connections": c.Int("concurrent_connections"),
		"os": c.Dict("os", s.Schema{
			"load": c.Dict("cpu.load_average", s.Schema{
				"1m":  c.Float("1m"),
				"5m":  c.Float("5m"),
				"15m": c.Float("15m"),
			}),
			"memory": c.Dict("mem", s.Schema{
				"total_in_bytes": c.Int("total_bytes"),
				"free_in_bytes":  c.Int("free_bytes"),
				"used_in_bytes":  c.Int("used_bytes"),
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
		}),
		"process": c.Dict("process", s.Schema{
			"event_loop_delay": c.Float("event_loop_delay_ms"),
			"memory": c.Dict("mem", s.Schema{
				"heap": s.Object{
					"total_in_bytes":    c.Int("heap_max_bytes"),
					"used_in_bytes":     c.Int("heap_used_bytes"),
					"external_in_bytes": c.Int("external_bytes"), // TODO: new field, must update monitoring-kibana template in ES x-pack plugin
					"size_limit":        c.Int("size_limit"),
				},
			}),
			"uptime_in_millis": c.Int("uptime_ms"),
		}),
		"requests": c.Dict("requests", s.Schema{
			"disconnects": c.Int("disconnects"),
			"total":       c.Int("total"),
		}),
		"response_times": c.Dict("response_times", s.Schema{
			"average": c.Float("avg_ms"),
			"max":     c.Float("max_ms"),
		}),
		"sockets": c.Dict("sockets", s.Schema{
			"http": c.Dict("http", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}),
			"https": c.Dict("https", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}),
		}),
		"kibana": c.Dict("kibana", s.Schema{
			"uuid":              c.Str("uuid"),
			"name":              c.Str("name"),
			"index":             c.Str("index"),
			"host":              c.Str("host"),
			"transport_address": c.Str("transport_address"),
			"version":           c.Str("version"),
			"snapshot":          c.Bool("snapshot"),
			"status":            c.Str("status"),
		}),
		"usage": c.Dict("usage", s.Schema{
			"index": c.Str("index"),
			"dashboard": c.Dict("dashboard", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"visualization": c.Dict("visualization", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"search": c.Dict("search", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"index_pattern": c.Dict("index_pattern", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"graph_workspace": c.Dict("graph_workspace", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"timelion_sheet": c.Dict("timelion_sheet", s.Schema{
				"total": s.Object{
					"count": c.Int("total"),
				},
			}, c.DictOptional),
			"xpack": c.Dict("xpack", s.Schema{
				"reporting": c.Dict("reporting", s.Schema{
					"available":    c.Bool("available"),
					"enabled":      c.Bool("enabled"),
					"browser_type": c.Str("browser_type"),
					"_all": s.Object{
						"count": c.Int("_all"),
					},
					"csv": c.Dict("csv", s.Schema{
						"available": c.Bool("available"),
						"total": s.Object{
							"count": c.Int("total"),
						},
					}, c.DictOptional),
					"printable_pdf": c.Dict("printable_pdf", s.Schema{
						"available": c.Bool("available"),
						"total": s.Object{
							"count": c.Int("total"),
						},
					}, c.DictOptional),
					"status": c.Dict("status", s.Schema{
						"completed":  c.Int("completed"),
						"failed":     c.Int("failed"),
						"processing": c.Int("processing"),
						"pending":    c.Int("pending"),
					}),
					"lastDay": c.Dict("lastDay", s.Schema{
						"_all": s.Object{
							"count": c.Int("_all"),
						},
						"csv": c.Dict("csv", s.Schema{
							"available": c.Bool("available"),
							"total": s.Object{
								"count": c.Int("total"),
							},
						}, c.DictOptional),
						"printable_pdf": c.Dict("printable_pdf", s.Schema{
							"available": c.Bool("available"),
							"total": s.Object{
								"count": c.Int("total"),
							},
						}, c.DictOptional),
						"status": c.Dict("status", s.Schema{
							"completed":  c.Int("completed"),
							"failed":     c.Int("failed"),
							"processing": c.Int("processing"),
							"pending":    c.Int("pending"),
						}),
					}, c.DictOptional),
					"last7Days": c.Dict("last7Days", s.Schema{
						"_all": s.Object{
							"count": c.Int("_all"),
						},
						"csv": c.Dict("csv", s.Schema{
							"available": c.Bool("available"),
							"total": s.Object{
								"count": c.Int("total"),
							},
						}, c.DictOptional),
						"printable_pdf": c.Dict("printable_pdf", s.Schema{
							"available": c.Bool("available"),
							"total": s.Object{
								"count": c.Int("total"),
							},
						}, c.DictOptional),
						"status": c.Dict("status", s.Schema{
							"completed":  c.Int("completed"),
							"failed":     c.Int("failed"),
							"processing": c.Int("processing"),
							"pending":    c.Int("pending"),
						}),
					}, c.DictOptional),
				}, c.DictOptional),
			}, c.DictOptional),
		}),
	}
)

func eventMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	kibanaStatsFields, err := schemaXPack.Apply(data)
	if err != nil {
		r.Error(err)
		return err
	}

	process := data["process"].(map[string]interface{})
	mem := process["mem"].(map[string]interface{})
	kibanaStatsFields.Put("process.memory.resident_set_size_in_bytes", mem["resident_set_size_bytes"].(int))

	timestamp := time.Now()
	kibanaStatsFields.Put("timestamp", timestamp)

	var event mb.Event
	event.RootFields = common.MapStr{
		"cluster_uuid": data["cluster_uuid"].(string),
		"timestamp":    timestamp,
		"interval_ms":  m.Module().Config().Period.Nanoseconds() / 1000 / 1000,
		"type":         "kibana_stats",
		"kibana_stats": kibanaStatsFields,
	}

	event.Index = helper.MakeMonitoringIndexName("kibana")
	r.Event(event)

	return nil
}
