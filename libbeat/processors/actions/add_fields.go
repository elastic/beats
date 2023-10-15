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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type addFields struct {
	fields    mapstr.M
	overwrite bool
}

// FieldsKey is the default target key for the add_fields processor.
const FieldsKey = "fields"

func init() {
	processors.RegisterPlugin("add_fields",
		checks.ConfigChecked(CreateAddFields,
			checks.RequireFields(FieldsKey),
			checks.AllowedFields(FieldsKey, "target", "when")))

	jsprocessor.RegisterPlugin("AddFields", CreateAddFields)
}

// CreateAddFields constructs an add_fields processor from config.
func CreateAddFields(c *conf.C) (beat.Processor, error) {
	config := struct {
		Fields mapstr.M `config:"fields" validate:"required"`
		Target *string  `config:"target"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the add_fields configuration: %w", err)
	}

	return makeFieldsProcessor(
		optTarget(config.Target, FieldsKey),
		config.Fields,
	), nil
}

// NewAddFields creates a new processor adding the given fields to events.
func NewAddFields(fields mapstr.M, overwrite bool) beat.Processor {
	return &addFields{fields: fields, overwrite: overwrite}
}

func (af *addFields) Run(event *beat.EventEditor) (dropped bool, err error) {
	if event == nil || len(af.fields) == 0 {
		return false, nil
	}

	if af.overwrite {
		event.DeepUpdate(af.fields)
	} else {
		event.DeepUpdateNoOverwrite(af.fields)
	}

	return false, nil
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

func makeFieldsProcessor(target string, fields mapstr.M) beat.Processor {
	if target != "" {
		fields = mapstr.M{
			target: fields,
		}
	}

	return NewAddFields(fields, true)
}
