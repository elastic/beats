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
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
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
}

func newDropFields(c *common.Config) (processors.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the drop_fields configuration: %s", err)
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

			regexpFields = append(regexpFields, match.MustCompile(field[1:len(field)-1]))
		}
	}

	f := &dropFields{Fields: config.Fields, IgnoreMissing: config.IgnoreMissing, RegexpFields: regexpFields}
	return f, nil
}

func (f *dropFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []error

	// remove exact match fields
	for _, field := range f.Fields {
		f.deleteField(event, field, &errs)
	}

	// remove fields contained in regexp expressions
	for _, regex := range f.RegexpFields {
		for _, field := range *event.Fields.FlattenKeys() {
			if regex.MatchString(field) {
				f.deleteField(event, field, &errs)
			}
		}
	}

	return event, multierr.Combine(errs...)
}

func (f *dropFields) deleteField(event *beat.Event, field string, errs *[]error) {
	if err := event.Delete(field); err != nil {
		if !f.IgnoreMissing || err != common.ErrKeyNotFound {
			*errs = append(*errs, errors.Wrapf(err, "failed to drop field [%v]", field))
		}
	}
}

func (f *dropFields) String() string {
	json, _ := json.Marshal(f)
	return "drop_fields=" + string(json)
}
