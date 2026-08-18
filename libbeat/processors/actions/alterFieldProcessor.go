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

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type alterFieldProcessor struct {
	Fields         []string
	Values         []string
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
		Values         []string `config:"values"`
	}{
		IgnoreMissing:  false,
		FailOnError:    true,
		AlterFullField: true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %s fields configuration: %w", processorName, err)
	}

	// Skip mandatory fields
	var configFields []string
	var lowerField string
	for _, readOnly := range processors.MandatoryExportedFields {
		readOnly = strings.ToLower(readOnly)
		for _, field := range config.Fields {
			// Skip fields that match "readOnly" or start with "readOnly."
			lowerField = strings.ToLower(field)
			if strings.HasPrefix(lowerField, readOnly+".") || lowerField == readOnly {
				continue
			}
			// Add fields that do not match "readOnly" criteria
			configFields = append(configFields, field)
		}
	}
	return &alterFieldProcessor{
		Fields:         configFields,
		IgnoreMissing:  config.IgnoreMissing,
		FailOnError:    config.FailOnError,
		processorName:  processorName,
		AlterFullField: config.AlterFullField,
		alterFunc:      alterFunc,
		Values:         config.Values,
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
		err := a.alterField(event, field)
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

	for _, valueKey := range a.Values {
		err := a.alterValue(event, valueKey)
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

func (a *alterFieldProcessor) alterField(event *beat.Event, field string) error {

	// modify all segments of the key
	var err error
	if a.AlterFullField {
		err = event.Fields.AlterPath(field, mapstr.CaseInsensitiveMode, a.alterFunc)
	} else {
		// modify only the last segment
		segmentCount := strings.Count(field, ".")
		err = event.Fields.AlterPath(field, mapstr.CaseInsensitiveMode, func(key string) (string, error) {
			if segmentCount > 0 {
				segmentCount--
				return key, nil
			}
			return a.alterFunc(key)
		})
	}

	return err
}

func (a *alterFieldProcessor) alterValue(event *beat.Event, valueKey string) error {
	value, err := event.GetValue(valueKey)
	if err != nil {
		return fmt.Errorf("could not fetch value for key: %s, Error: %w", valueKey, err)
	}

	if v, ok := value.(string); ok {
		err = event.Delete(valueKey)
		if err != nil {
			return fmt.Errorf("could not delete key: %s,  %w", v, err)
		}

		v, err = a.alterFunc(v)
		if err != nil {
			return fmt.Errorf("could not alter %s successfully, %w", v, err)
		}

		_, err = event.PutValue(valueKey, v)
		if err != nil {
			return fmt.Errorf("could not put value: %s: %v, %w", valueKey, v, err)
		}
	} else {
		return fmt.Errorf("value of key %q is not a string", valueKey)
	}

	return nil
}
