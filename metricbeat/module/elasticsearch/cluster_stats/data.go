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

package cluster_stats

import (
	"encoding/json"

	"github.com/elastic/beats/metricbeat/helper/elastic"

	"github.com/elastic/beats/libbeat/common"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"status": c.Str("status"),
		"nodes": c.Dict("nodes", s.Schema{
			"count":  c.Int("count.total"),
			"master": c.Int("count.master"),
			"data":   c.Int("count.data"),
		}),
		"indices": c.Dict("indices", s.Schema{
			"total": c.Int("count"),
			"shards": c.Dict("shards", s.Schema{
				"count":     c.Int("total"),
				"primaries": c.Int("primaries"),
			}),
			"fielddata": c.Dict("fielddata", s.Schema{
				"memory": s.Object{
					"bytes": c.Int("memory_size_in_bytes"),
				},
			}),
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

	metricSetFields, err := schema.Apply(data)
	if err != nil {
		r.Error(err)
		return err
	}

	clusterName, ok := data["cluster_name"]
	if !ok {
		return elastic.ReportErrorForMissingField("cluster_name", elastic.Elasticsearch, r)
	}

	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", "elasticsearch")

	event.ModuleFields = common.MapStr{}
	event.ModuleFields.Put("cluster.name", clusterName)
	clusterUUID, ok := data["cluster_uuid"]
	if ok {
		event.ModuleFields.Put("cluster.id", clusterUUID)
	}

	event.MetricSetFields = metricSetFields

	r.Event(event)
	return nil
}
