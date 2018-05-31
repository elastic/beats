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
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

var (
	schema = s.Schema{
		// This is all shard information and should be linked to elasticsearch.shard.*
		// as soon as field aliases are available.
		"id":      c.Str("id"),
		"type":    c.Str("type"),
		"primary": c.Str("primary"),
		"stage":   c.Str("stage"),

		// As soon as we have field alias feature available, source and target should
		// link to elasticsearch.node.* as it's not specific information.
		"source": c.Dict("source", s.Schema{
			"id":   c.Str("id"),
			"host": c.Str("host"),
			"name": c.Str("name"),
		}),
		"target": c.Dict("target", s.Schema{
			"id":   c.Str("id"),
			"host": c.Str("host"),
			"name": c.Str("name"),
		}),
	}
)

func eventsMapping(r mb.ReporterV2, info elasticsearch.Info, content []byte) error {

	var data map[string]map[string][]map[string]interface{}

	err := json.Unmarshal(content, &data)
	if err != nil {
		r.Error(err)
		return err
	}

	for indexName, d := range data {
		for _, data := range d["shards"] {
			event := mb.Event{}
			event.ModuleFields = common.MapStr{}
			event.ModuleFields.Put("index.name", indexName)
			event.MetricSetFields, err = schema.Apply(data)
			event.RootFields = common.MapStr{}
			event.RootFields.Put("service.name", "elasticsearch")
			r.Event(event)
		}
	}
	return nil
}
