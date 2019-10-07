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

package index_summary

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		"primaries": c.Dict("primaries", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
				"memory": s.Object{
					"bytes": c.Int("memory_in_bytes"),
				},
			}),
		}),
		"total": c.Dict("total", s.Schema{
			"docs": c.Dict("docs", s.Schema{
				"count":   c.Int("count"),
				"deleted": c.Int("deleted"),
			}),
			"store": c.Dict("store", s.Schema{
				"size": s.Object{
					"bytes": c.Int("size_in_bytes"),
				},
			}),
			"segments": c.Dict("segments", s.Schema{
				"count": c.Int("count"),
				"memory": s.Object{
					"bytes": c.Int("memory_in_bytes"),
				},
			}),
		}),
	}
)

func eventMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {
	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", elasticsearch.ModuleName)

	event.ModuleFields = common.MapStr{}
	event.ModuleFields.Put("cluster.name", info.ClusterName)
	event.ModuleFields.Put("cluster.id", info.ClusterID)

	var all struct {
		Data map[string]interface{} `json:"_all"`
	}

	err := json.Unmarshal(content, &all)
	if err != nil {
		return errors.Wrap(err, "failure parsing Elasticsearch Stats API response")
	}

	fields, err := schema.Apply(all.Data, s.FailOnRequired)
	if err != nil {
		return errors.Wrap(err, "failure applying stats schema")
	}

	event.MetricSetFields = fields

	r.Event(event)
	return nil
}
