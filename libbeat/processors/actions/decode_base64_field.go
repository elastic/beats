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
	"strings"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/processors"
	"github.com/menderesk/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/menderesk/beats/v7/libbeat/processors/script/javascript/module/processor"
)

const (
	processorName = "decode_base64_field"
)

type decodeBase64Field struct {
	config base64Config
	log    *logp.Logger
}

type base64Config struct {
	Field         fromTo `config:"field"`
	IgnoreMissing bool   `config:"ignore_missing"`
	FailOnError   bool   `config:"fail_on_error"`
}

func init() {
	processors.RegisterPlugin(processorName,
		checks.ConfigChecked(NewDecodeBase64Field,
			checks.RequireFields("field"),
			checks.AllowedFields("field", "when", "ignore_missing", "fail_on_error")))
	jsprocessor.RegisterPlugin("DecodeBase64Field", NewDecodeBase64Field)
}

// NewDecodeBase64Field construct a new decode_base64_field processor.
func NewDecodeBase64Field(c *common.Config) (processors.Processor, error) {
	config := base64Config{
		IgnoreMissing: false,
		FailOnError:   true,
	}

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the %s configuration: %s", processorName, err)
	}

	return &decodeBase64Field{
		config: config,
		log:    logp.NewLogger(processorName),
	}, nil
}

func (f *decodeBase64Field) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	// Creates a copy of the event to revert in case of failure
	if f.config.FailOnError {
		backup = event.Clone()
	}

	err := f.decodeField(event)
	if err != nil {
		errMsg := fmt.Errorf("failed to decode base64 fields in processor: %v", err)
		f.log.Debug(errMsg.Error())
		if f.config.FailOnError {
			event = backup
			event.PutValue("error.message", errMsg.Error())
			return event, err
		}
	}
	return event, nil
}

func (f decodeBase64Field) String() string {
	return fmt.Sprintf("%s=%+v", processorName, f.config.Field)
}

func (f *decodeBase64Field) decodeField(event *beat.Event) error {
	value, err := event.GetValue(f.config.Field.From)
	if err != nil {
		if f.config.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("could not fetch base64 value for key: %s, Error: %v", f.config.Field.From, err)
	}

	base64String, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid type for `from`, expecting a string received %T", value)
	}

	decodedData, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(base64String, "="))
	if err != nil {
		return fmt.Errorf("error trying to decode %s: %v", base64String, err)
	}

	target := f.config.Field.To
	// If to is empty
	if f.config.Field.To == "" || f.config.Field.From == f.config.Field.To {
		target = f.config.Field.From
	}

	if _, err = event.PutValue(target, string(decodedData)); err != nil {
		return fmt.Errorf("could not put value: %s: %v, %v", decodedData, target, err)
	}

	return nil
}
