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

package state

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
		"management": c.Dict("management", s.Schema{
			"enabled": c.Bool("enabled"),
		}),
		"module": c.Dict("module", s.Schema{
			"count": c.Int("count"),
		}),
		"output": c.Dict("output", s.Schema{
			"name": c.Str("name"),
		}),
		"queue": c.Dict("queue", s.Schema{
			"name": c.Str("name"),
		}),
	}
)

func eventMapping(r mb.ReporterV2, info beat.Info, content []byte) error {
	var event mb.Event
	event.RootFields = common.MapStr{}
	event.RootFields.Put("service", common.MapStr{
		"id":   info.UUID,
		"name": info.Name,
	})

	event.Service = info.Beat

	var data map[string]interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return errors.Wrap(err, "failure parsing Beat's State API response")
	}

	event.MetricSetFields, err = schema.Apply(data)
	if err != nil {
		return errors.Wrap(err, "failure to apply state schema")
	}

	r.Event(event)
	return nil
}
