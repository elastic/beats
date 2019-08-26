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

package connections

import (
	"encoding/json"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id":   c.Str("server_id"),
			"time": c.Str("now"),
		},
	}
	connectionsSchema = s.Schema{
		"total": c.Int("total"),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var event mb.Event
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return errors.Wrap(err, "failure parsing NATS connections API response")
	}
	event.MetricSetFields, err = connectionsSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying connections schema")

	}

	event.ModuleFields, err = moduleSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying module schema")
	}
	r.Event(event)
	return nil
}
