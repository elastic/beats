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
	"strings"

	"errors"

	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/libbeat/common/match"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type dropFields struct {
	Fields        []string
	RegexpFields  []match.Matcher
	IgnoreMissing bool
}

func init() {
	processors.RegisterPlugin("drop_fields",
		checks.ConfigChecked(newDropFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "when", "ignore_missing")))

	jsprocessor.RegisterPlugin("DropFields", newDropFields)
}

func newDropFields(c *conf.C) (beat.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the drop_fields configuration: %w", err)
	}

	/* remove read only fields */
	// TODO: Is this implementation used? If so, there's a fix needed in removal of exported fields
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if readOnly == field {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	// Parse regexp containing fields and removes them from initial config
	regexpFields := make([]match.Matcher, 0)
	for i := len(config.Fields) - 1; i >= 0; i-- {
		field := config.Fields[i]
		if strings.HasPrefix(field, "/") && strings.HasSuffix(field, "/") && len(field) > 2 {
			config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)

			matcher, err := match.Compile(field[1 : len(field)-1])
			if err != nil {
				return nil, fmt.Errorf("wrong configuration in drop_fields[%d]=%s. %w", i, field, err)
			}

			regexpFields = append(regexpFields, matcher)
		}
	}

	f := &dropFields{Fields: config.Fields, IgnoreMissing: config.IgnoreMissing, RegexpFields: regexpFields}
	return f, nil
}

func (f *dropFields) Run(event *beat.EventEditor) (dropped bool, err error) {
	var errs []error

	droppedKeys := make(map[string]struct{})
	// remove exact match fields
	for _, field := range f.Fields {
		if f.checkAlreadyDropped(droppedKeys, field) {
			continue
		}
		droppedKeys[field] = struct{}{}
		f.deleteField(event, field, &errs)
	}

	// remove fields contained in regexp expressions
	for _, field := range event.FlattenKeys() {
		if f.checkAlreadyDropped(droppedKeys, field) {
			continue
		}
		for _, regex := range f.RegexpFields {
			if !regex.MatchString(field) {
				continue
			}
			droppedKeys[field] = struct{}{}
			f.deleteField(event, field, &errs)
		}
	}

	return false, multierr.Combine(errs...)
}

func (f *dropFields) checkAlreadyDropped(droppedKeys map[string]struct{}, key string) bool {
	_, dropped := droppedKeys[key]
	if dropped {
		return true
	}
	for droppedKey := range droppedKeys {
		if strings.HasPrefix(key, droppedKey) {
			return true
		}
	}

	return false
}

func (f *dropFields) deleteField(event *beat.EventEditor, field string, errs *[]error) {
	if err := event.Delete(field); err != nil {
		if !f.IgnoreMissing || !errors.Is(err, mapstr.ErrKeyNotFound) {
			*errs = append(*errs, fmt.Errorf("failed to drop field [%v], error: %w", field, err))
		}
	}
}

func (f *dropFields) String() string {
	json, _ := json.Marshal(f)
	return "drop_fields=" + string(json)
}
