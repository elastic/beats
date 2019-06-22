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
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/beat"
)

var (
	schema = s.Schema{
		"uptime": c.Dict("beat.info.uptime", s.Schema{
			"ms": c.Int("ms"),
		}),
		"runtime": c.Dict("beat.runtime", s.Schema{
			"goroutines": c.Int("goroutines"),
		}, c.DictOptional),
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
		}),
	}
)

func eventMapping(r mb.ReporterV2, info beat.Info, content []byte) error {
	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service.name", beat.ModuleName)

	event.ModuleFields = common.MapStr{}
	event.ModuleFields.Put("id", info.UUID)
	event.ModuleFields.Put("type", info.Beat)

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's Stats API response")
	}

	event.MetricSetFields, err = schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure to apply stats schema")
	}

	r.Event(event)
	return nil
}
