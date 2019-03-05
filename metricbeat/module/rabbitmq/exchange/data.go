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

package exchange

import (
	"encoding/json"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"name":        c.Str("name"),
		"vhost":       c.Str("vhost"),
		"type":        c.Str("type"),
		"durable":     c.Bool("durable"),
		"auto_delete": c.Bool("auto_delete"),
		"internal":    c.Bool("internal"),
		"arguments":   c.Dict("arguments", s.Schema{}),
		"user":        c.Str("user_who_performed_action", s.Optional),
		"messages": c.Dict("message_stats", s.Schema{
			"publish_in": s.Object{
				"count": c.Int("publish_in", s.Optional),
				"details": c.Dict("publish_in_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
			"publish_out": s.Object{
				"count": c.Int("publish_out", s.Optional),
				"details": c.Dict("publish_out_details", s.Schema{
					"rate": c.Float("rate"),
				}, c.DictOptional),
			},
		}, c.DictOptional),
	}
)

func eventsMapping(content []byte, r mb.ReporterV2) {
	var exchanges []map[string]interface{}
	err := json.Unmarshal(content, &exchanges)
	if err != nil {
		logp.Err("Error: %+v", err)
		r.Error(err)
		return
	}

	var errors multierror.Errors

	for _, exchange := range exchanges {
		err := eventMapping(exchange, r)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		r.Error(errors.Err())
	}
}

func eventMapping(exchange map[string]interface{}, r mb.ReporterV2) error {
	fields, err := schema.Apply(exchange)
	if err != nil {
		return err
	}

	rootFields := common.MapStr{}
	if v, err := fields.GetValue("user"); err == nil {
		rootFields.Put("user.name", v)
		fields.Delete("user")
	}

	moduleFields := common.MapStr{}
	if v, err := fields.GetValue("vhost"); err == nil {
		moduleFields.Put("vhost", v)
		fields.Delete("vhost")
	}

	event := mb.Event{
		MetricSetFields: fields,
		RootFields:      rootFields,
		ModuleFields:    moduleFields,
	}
	r.Event(event)
	return nil
}
