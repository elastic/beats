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

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
)

type dropFields struct {
	Fields        []string
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
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if readOnly == field {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	f := &dropFields{Fields: config.Fields, IgnoreMissing: config.IgnoreMissing}
	return f, nil
}

func (f *dropFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []error

	for _, field := range f.Fields {
		if err := event.Delete(field); err != nil {
			if f.IgnoreMissing && err == common.ErrKeyNotFound {
				continue
			}
			errs = append(errs, errors.Wrapf(err, "failed to drop field [%v]", field))
		}
	}

	return event, multierr.Combine(errs...)
}

func (f *dropFields) String() string {
	json, _ := json.Marshal(f)
	return "drop_fields=" + string(json)
}
