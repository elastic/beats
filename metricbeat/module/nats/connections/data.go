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

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
)

var (
	statsSchema = s.Schema{
		"server_id": c.Str("server_id"),
		"now":       c.Str("now"),
		"total":     c.Int("total"),
	}
)

func eventMapping(r mb.ReporterV2, content []byte) error {
	var event common.MapStr
	var inInterface map[string]interface{}

	err := json.Unmarshal(content, &inInterface)
	if err != nil {
		err = errors.Wrap(err, "failure parsing Nats connections API response")
		r.Error(err)
		return err
	}
	event, err = statsSchema.Apply(inInterface)
	if err != nil {
		err = errors.Wrap(err, "failure applying index schema")
		r.Error(err)
		return err
	}

	r.Event(mb.Event{MetricSetFields: event})
	return nil
}
