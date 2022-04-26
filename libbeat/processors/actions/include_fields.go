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
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type includeFields struct {
	Fields []string
}

func init() {
	processors.RegisterPlugin("include_fields",
		checks.ConfigChecked(newIncludeFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "when")))
}

func newIncludeFields(c *common.Config) (processors.Processor, error) {
	config := struct {
		Fields []string `config:"fields"`
	}{}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the include_fields configuration: %s", err)
	}

	/* add read only fields if they are not yet */
	for _, readOnly := range processors.MandatoryExportedFields {
		found := false
		for _, field := range config.Fields {
			if readOnly == field {
				found = true
			}
		}
		if !found {
			config.Fields = append(config.Fields, readOnly)
		}
	}

	f := &includeFields{Fields: config.Fields}
	return f, nil
}

func (f *includeFields) Run(event *beat.Event) (*beat.Event, error) {
	filtered := mapstr.M{}
	var errs []string

	for _, field := range f.Fields {
		v, err := event.GetValue(field)
		if err == nil {
			_, err = filtered.Put(field, v)
		}

		// Ignore ErrKeyNotFound errors
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			errs = append(errs, err.Error())
		}
	}

	event.Fields = filtered
	if len(errs) > 0 {
		return event, fmt.Errorf(strings.Join(errs, ", "))
	}
	return event, nil
}

func (f *includeFields) String() string {
	return "include_fields=" + strings.Join(f.Fields, ", ")
}
