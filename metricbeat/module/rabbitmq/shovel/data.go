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

package shovel

import (
	"encoding/json"
	"fmt"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	schema = s.Schema{
		"name":  c.Str("name"),
		"vhost": c.Str("vhost"),
		"type":  c.Str("type"),
		"node":  c.Str("node"),
		"state": c.Str("state"),
	}
)

func eventsMapping(content []byte, r mb.ReporterV2) error {
	var shovels []map[string]interface{}
	err := json.Unmarshal(content, &shovels)
	if err != nil {
		return fmt.Errorf("error in mapping: %w", err)
	}

	for _, shovel := range shovels {
		evt := eventMapping(shovel)
		r.Event(evt)
	}

	return nil
}

func eventMapping(shovel map[string]interface{}) mb.Event {
	fields, _ := schema.Apply(shovel)

	moduleFields := mapstr.M{}
	if v, err := fields.GetValue("vhost"); err == nil {
		_, _ = moduleFields.Put("vhost", v)
		_ = fields.Delete("vhost")
	}

	if v, err := fields.GetValue("node"); err == nil {
		_, _ = moduleFields.Put("node.name", v)
		_ = fields.Delete("node")
	}

	event := mb.Event{
		MetricSetFields: fields,
		ModuleFields:    moduleFields,
	}
	return event
}
