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

package actions

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type addFields struct {
	fields common.MapStr
	shared bool
}

// FieldsKey is the default target key for the add_fields processor.
const FieldsKey = "fields"

func init() {
	processors.RegisterPlugin("add_fields",
		configChecked(CreateAddFields,
			requireFields(FieldsKey),
			allowedFields(FieldsKey, "target", "when")))
}

// CreateAddFields constructs an add_fields processor from config.
func CreateAddFields(c *common.Config) (processors.Processor, error) {
	config := struct {
		Fields common.MapStr `config:"fields" validate:"required"`
		Target *string       `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %s", err)
	}

	return makeFieldsProcessor(
		optTarget(config.Target, FieldsKey),
		config.Fields,
		true,
	), nil
}

// NewAddFields creates a new processor adding the given fields to events.
// Set `shared` true if there is the chance of labels being changed/modified by
// subsequent processors.
func NewAddFields(fields common.MapStr, shared bool) processors.Processor {
	return &addFields{fields: fields, shared: shared}
}

func (af *addFields) Run(event *beat.Event) (*beat.Event, error) {
	fields := af.fields
	if af.shared {
		fields = fields.Clone()
	}

	event.Fields.DeepUpdate(fields)
	return event, nil
}

func (af *addFields) String() string {
	s, _ := json.Marshal(af.fields)
	return fmt.Sprintf("add_fields=%s", s)
}

func optTarget(opt *string, def string) string {
	if opt == nil {
		return def
	}
	return *opt
}

func makeFieldsProcessor(target string, fields common.MapStr, shared bool) processors.Processor {
	if target != "" {
		fields = common.MapStr{
			target: fields,
		}
	}

	return NewAddFields(fields, shared)
}
