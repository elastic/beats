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

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
)

type copyFields struct {
	config copyFieldsConfig
}

type copyFieldsConfig struct {
	Fields        []fromTo `config:"fields"`
	IgnoreMissing bool     `config:"ignore_missing"`
	FailOnError   bool     `config:"fail_on_error"`
}

func init() {
	processors.RegisterPlugin("copy_fields",
		checks.ConfigChecked(NewCopyFields,
			checks.RequireFields("fields"),
		),
	)
}

// NewCopyFields returns a new copy_fields processor.
func NewCopyFields(c *common.Config) (processors.Processor, error) {
	config := copyFieldsConfig{
		IgnoreMissing: false,
		FailOnError:   true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of copy processor: %s", err)
	}

	f := &copyFields{
		config: config,
	}
	return f, nil
}

func (f *copyFields) Run(event *beat.Event) (*beat.Event, error) {
	var backup common.MapStr
	if f.config.FailOnError {
		backup = event.Fields.Clone()
	}

	for _, field := range f.config.Fields {
		err := f.copyField(field.From, field.To, event.Fields)
		if err != nil && f.config.FailOnError {
			errMsg := fmt.Errorf("Failed to copy fields in copy_fields processor: %s", err)
			logp.Debug("copy_fields", errMsg.Error())
			event.Fields = backup
			event.PutValue("error.message", errMsg.Error())
			return event, err
		}
	}

	return event, nil
}

func (f *copyFields) copyField(from string, to string, fields common.MapStr) error {
	exists, _ := fields.HasKey(to)
	if exists {
		return fmt.Errorf("target field %s already exists, drop or rename this field first", to)
	}

	value, err := fields.GetValue(from)
	if err != nil {
		if f.config.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %s", from, err)
	}

	_, err = fields.Put(to, value)
	if err != nil {
		return fmt.Errorf("could not copy value to %s: %v, %+v", to, value, err)
	}
	return nil
}

func (f *copyFields) String() string {
	return "copy_fields=" + fmt.Sprintf("%+v", f.config.Fields)
}
