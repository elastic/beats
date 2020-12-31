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

package routes

import (
	"encoding/json"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/nats/util"
)

var (
	moduleSchema = s.Schema{
		"server": s.Object{
			"id":   c.Str("server_id"),
			"time": c.Str("now"),
		},
	}
	routesSchema = s.Schema{
		"total": c.Int("num_routes"),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		return errors.Wrap(err, "failure parsing Nats routes API response")
	}
	metricSetFields, err := routesSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying routes schema")
	}

	moduleFields, err := moduleSchema.Apply(inInterface)
	if err != nil {
		return errors.Wrap(err, "failure applying module schema")
	}
	timestamp, err := util.GetNatsTimestamp(moduleFields)
	if err != nil {
		errors.Wrap(err, "failure parsing server timestamp")
	}
	event := mb.Event{
		MetricSetFields: metricSetFields,
		ModuleFields:    moduleFields,
		Timestamp:       timestamp,
	}
	r.Event(event)
	return nil
}
