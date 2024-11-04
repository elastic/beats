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
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type alterFieldProcessor struct {
	Fields         []string
	IgnoreMissing  bool
	FailOnError    bool
	AlterFullField bool

	processorName string
	alterFunc     mapstr.AlterFunc
}

// NewAlterFieldProcessor is an umbrella method for processing events based on provided fields. Such as converting event keys to uppercase/lowercase
func NewAlterFieldProcessor(c *conf.C, processorName string, alterFunc mapstr.AlterFunc) (beat.Processor, error) {
	config := struct {
		Fields         []string `config:"fields"`
		IgnoreMissing  bool     `config:"ignore_missing"`
		FailOnError    bool     `config:"fail_on_error"`
		AlterFullField bool     `config:"alter_full_field"`
	}{
		IgnoreMissing:  false,
		FailOnError:    true,
		AlterFullField: true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %s fields configuration: %w", processorName, err)
	}

	// Skip mandatory fields
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if field == readOnly {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}
	return &alterFieldProcessor{
		Fields:         config.Fields,
		IgnoreMissing:  config.IgnoreMissing,
		FailOnError:    config.FailOnError,
		processorName:  processorName,
		AlterFullField: config.AlterFullField,
		alterFunc:      alterFunc,
	}, nil

}

func (a *alterFieldProcessor) String() string {
	return fmt.Sprintf("%s fields=%+v", a.processorName, *a)
}

func (a *alterFieldProcessor) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if a.FailOnError {
		backup = event.Clone()
	}

	for _, field := range a.Fields {
		err := a.alter(event, field)
		if err != nil {
			if a.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
				continue
			}
			if a.FailOnError {
				event = backup
				_, _ = event.PutValue("error.message", err.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (a *alterFieldProcessor) alter(event *beat.Event, field string) error {

	// modify all segments of the key
	if a.AlterFullField {
		err := event.Fields.AlterPath(field, mapstr.CaseInsensitiveMode, a.alterFunc)
		if err != nil {
			return err
		}
	} else {
		// modify only the last segment
		segmentCount := strings.Count(field, ".")
		err := event.Fields.AlterPath(field, mapstr.CaseInsensitiveMode, func(key string) (string, error) {
			if segmentCount > 0 {
				segmentCount--
				return key, nil
			}
			return a.alterFunc(key)
		})
		if err != nil {
			return err
		}
	}

	return nil
}
