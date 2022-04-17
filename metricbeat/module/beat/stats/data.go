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

	"github.com/menderesk/beats/v7/metricbeat/helper/elastic"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	s "github.com/menderesk/beats/v7/libbeat/common/schema"
	c "github.com/menderesk/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/beat"
)

var (
	schema = s.Schema{
		"cgroup":     c.Ifc("beat.cgroup"),
		"system":     c.Ifc("system"),
		"apm_server": c.Ifc("apm-server"),
		"cpu":        c.Ifc("beat.cpu"),
		"info":       c.Ifc("beat.info"),
		"uptime": c.Dict("beat.info.uptime", s.Schema{
			"ms": c.Int("ms"),
		}),
		"runtime": c.Dict("beat.runtime", s.Schema{
			"goroutines": c.Int("goroutines"),
		}, c.DictOptional),
		"handles": c.Dict("beat.handles", s.Schema{
			"limit": c.Dict("limit", s.Schema{
				"hard": c.Int("hard"),
				"soft": c.Int("soft"),
			}),
			"open": c.Int("open"),
		}),
		"libbeat": c.Dict("libbeat", s.Schema{
			"output": c.Dict("output", s.Schema{
				"type": c.Str("type"),
				"events": c.Dict("events", s.Schema{
					"acked":      c.Int("acked"),
					"active":     c.Int("active"),
					"batches":    c.Int("batches"),
					"dropped":    c.Int("dropped"),
					"duplicates": c.Int("duplicates"),
					"failed":     c.Int("failed"),
					"toomany":    c.Int("toomany"),
					"total":      c.Int("total"),
				}),
				"read": c.Dict("read", s.Schema{
					"bytes":  c.Int("bytes"),
					"errors": c.Int("errors"),
				}),
				"write": c.Dict("write", s.Schema{
					"bytes":  c.Int("bytes"),
					"errors": c.Int("errors"),
				}),
			}),
			"pipeline": c.Dict("pipeline", s.Schema{
				"clients": c.Int("clients"),
				"queue": c.Dict("queue", s.Schema{
					"acked": c.Int("acked"),
				}),
				"events": c.Dict("events", s.Schema{
					"active":    c.Int("active"),
					"dropped":   c.Int("dropped"),
					"failed":    c.Int("failed"),
					"filtered":  c.Int("filtered"),
					"published": c.Int("published"),
					"retry":     c.Int("retry"),
					"total":     c.Int("total"),
				}),
			}),
			"config": c.Dict("config", s.Schema{
				"running": c.Int("module.running"),
				"starts":  c.Int("module.starts"),
				"stops":   c.Int("module.stops"),
				"reloads": c.Int("reloads"),
			}),
		}),
		"state": c.Dict("metricbeat.beat.state", s.Schema{
			"events":   c.Int("events"),
			"failures": c.Int("failures"),
			"success":  c.Int("success"),
		}),
		"memstats": c.Dict("beat.memstats", s.Schema{
			"gc_next": c.Int("gc_next"),
			"memory": s.Object{
				"alloc": c.Int("memory_alloc"),
				"total": c.Int("memory_total"),
			},
			"rss": c.Int("rss"),
		}),
	}
)

func eventMapping(r mb.ReporterV2, info beat.Info, clusterUUID string, content []byte, isXpack bool) error {
	event := mb.Event{
		RootFields:      common.MapStr{},
		ModuleFields:    common.MapStr{},
		MetricSetFields: common.MapStr{},
	}
	event.RootFields.Put("service.name", beat.ModuleName)

	event.ModuleFields.Put("id", info.UUID)
	event.ModuleFields.Put("type", info.Beat)

	if clusterUUID != "" {
		event.ModuleFields.Put("elasticsearch.cluster.id", clusterUUID)
	}

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's Stats API response")
	}

	event.MetricSetFields, _ = schema.Apply(data)
	event.MetricSetFields.Put("beat", common.MapStr{
		"name":    info.Name,
		"host":    info.Hostname,
		"type":    info.Beat,
		"uuid":    info.UUID,
		"version": info.Version,
	})

	// xpack.enabled in config using standalone metricbeat writes to `.monitoring` instead of `metricbeat-*`
	// When using Agent, the index name is overwritten anyways.
	if isXpack {
		index := elastic.MakeXPackMonitoringIndexName(elastic.Beats)
		event.Index = index
	}

	r.Event(event)
	return nil
}
