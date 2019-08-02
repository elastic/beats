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
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
)

const (
	processorName = "decode_base64_field"
)

type decodeBase64Field struct {
	log *logp.Logger

	config base64Config
}

type base64Config struct {
	Field         fromTo `config:"field"`
	IgnoreMissing bool   `config:"ignore_missing"`
	FailOnError   bool   `config:"fail_on_error"`
}

var (
	defaultBase64Config = base64Config{
		IgnoreMissing: false,
		FailOnError:   true,
	}
)

func init() {
	processors.RegisterPlugin(processorName,
		checks.ConfigChecked(NewDecodeBase64Field,
			checks.RequireFields("field"),
			checks.AllowedFields("field", "when")))
}

// NewDecodeBase64Field construct a new decode_base64_field processor.
func NewDecodeBase64Field(c *common.Config) (processors.Processor, error) {
	config := defaultBase64Config

	log := logp.NewLogger(processorName)

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the %s configuration: %s", processorName, err)
	}

	return &decodeBase64Field{
		log:    log,
		config: config,
	}, nil
}

func (f *decodeBase64Field) Run(event *beat.Event) (*beat.Event, error) {
	var backup common.MapStr
	// Creates a copy of the event to revert in case of failure
	if f.config.FailOnError {
		backup = event.Fields.Clone()
	}

	err := f.decodeField(f.config.Field.From, f.config.Field.To, event.Fields)
	if err != nil && f.config.FailOnError {
		errMsg := fmt.Errorf("failed to decode base64 fields in processor: %v", err)
		f.log.Debug("decode base64", errMsg.Error())
		event.Fields = backup
		_, _ = event.PutValue("error.message", errMsg.Error())
		return event, err
	}

	return event, nil
}

func (f decodeBase64Field) String() string {
	return fmt.Sprintf("%s=%+v", processorName, f.config.Field)
}

func (f *decodeBase64Field) decodeField(from string, to string, fields common.MapStr) error {
	value, err := fields.GetValue(from)
	if err != nil {
		// Ignore ErrKeyNotFound errors
		if f.config.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %s", from, err)
	}

	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for `from`, expecting a string received %T", value)
	}

	decodedData, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return fmt.Errorf("error trying to unmarshal %s: %v", text, err)
	}

	field := to
	// If to is empty
	if to == "" || from == to {
		// Deletion must happen first to support cases where a becomes a.b
		if err = fields.Delete(from); err != nil {
			return fmt.Errorf("could not delete key: %s,  %+v", from, err)
		}

		field = from
	}

	if _, err = fields.Put(field, string(decodedData)); err != nil {
		return fmt.Errorf("could not put value: %s: %v, %v", decodedData, field, err)
	}

	return nil
}
