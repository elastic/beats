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

package index_recovery

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		// This is all shard information and should be linked to elasticsearch.shard.*
		// as soon as field aliases are available.
		"id":      c.Int("id"),
		"type":    c.Str("type"),
		"primary": c.Bool("primary"),
		"stage":   c.Str("stage"),

		// As soon as we have field alias feature available, source and target should
		// link to elasticsearch.node.* as it's not specific information.
		"source": c.Dict("source", s.Schema{
			"id":   c.Str("id", s.Optional),
			"host": c.Str("host", s.Optional),
			"name": c.Str("name", s.Optional),
		}),
		"target": c.Dict("target", s.Schema{
			"id":   c.Str("id", s.Optional),
			"host": c.Str("host", s.Optional),
			"name": c.Str("name", s.Optional),
		}),
	}
)

func eventsMapping(r mb.ReporterV2, content []byte) error {

	var data map[string]map[string][]map[string]interface{}

	err := json.Unmarshal(content, &data)
	if err != nil {
		return err
	}

	for indexName, d := range data {
		shards, ok := d["shards"]
		if !ok {
			continue
		}
		for _, data := range shards {
			event := mb.Event{}
			event.ModuleFields = common.MapStr{}
			event.MetricSetFields, _ = schema.Apply(data)
			event.ModuleFields.Put("index.name", indexName)
			event.RootFields = common.MapStr{}
			event.RootFields.Put("service.name", "elasticsearch")
			r.Event(event)
		}
	}
	return nil
}
