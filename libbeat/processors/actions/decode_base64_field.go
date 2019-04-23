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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/pkg/errors"
)

const (
	processorName = "decode_base64_field"
)

type decodeBase64Fields struct {
	log *logp.Logger

	field  string
	target *string
}

type base64Config struct {
	Field  string  `config:"field"`
	Target *string `config:"target"`
}

var (
	defaultBase64Config = base64Config{}
)

func init() {
	processors.RegisterPlugin(processorName,
		configChecked(NewDecodeBase64Field,
			requireFields("field"),
			allowedFields("field", "target")))
}

// NewDecodeBase64Field construct a new decode_base64_field processor.
func NewDecodeBase64Field(c *common.Config) (processors.Processor, error) {
	config := defaultBase64Config

	log := logp.NewLogger(processorName)

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the %s configuration: %s", processorName, err)
	}

	return &decodeBase64Fields{
		log:    log,
		field:  config.Field,
		target: config.Target,
	}, nil
}

func (f *decodeBase64Fields) Run(event *beat.Event) (*beat.Event, error) {
	data, err := event.GetValue(f.field)
	if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
		return event, fmt.Errorf("error trying to GetValue for field : %s in event : %v : %v", f.field, event, err)
	}

	text, ok := data.(string)
	if !ok {
		// ignore non string fields when unmarshaling
		return event, nil
	}

	decodeData, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return event, fmt.Errorf("error trying to unmarshal %s: %v", text, err)
	}

	target := f.field
	if f.target != nil {
		target = *f.target
	}

	if target != "" {
		if _, err = event.PutValue(target, string(decodeData)); err != nil {
			return event, fmt.Errorf("error trying to Put value %v for field : %s: %v", decodeData, f.field, err)
		}
	}

	return event, nil
}

func (f decodeBase64Fields) String() string {
	return fmt.Sprintf("%s=%s", processorName, f.field)
}
