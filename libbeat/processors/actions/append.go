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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type appendProcessor struct {
	config appendProcessorConfig
	logger *logp.Logger
}

type appendProcessorConfig struct {
	Fields            []string      `config:"fields"`
	TargetField       string        `config:"target_field"`
	Values            []interface{} `config:"values"`
	IgnoreMissing     bool          `config:"ignore_missing"`
	IgnoreEmptyValues bool          `config:"ignore_empty_values"`
	FailOnError       bool          `config:"fail_on_error"`
	AllowDuplicate    bool          `config:"allow_duplicate"`
}

func init() {
	processors.RegisterPlugin("append",
		checks.ConfigChecked(NewAppendProcessor,
			checks.RequireFields("target_field"),
		),
	)
	jsprocessor.RegisterPlugin("AppendProcessor", NewAppendProcessor)
}

// NewAppendProcessor returns a new append processor.
func NewAppendProcessor(c *conf.C) (beat.Processor, error) {
	config := appendProcessorConfig{
		IgnoreMissing:     false,
		IgnoreEmptyValues: false,
		FailOnError:       true,
		AllowDuplicate:    true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of append processor: %w", err)
	}

	f := &appendProcessor{
		config: config,
		logger: logp.NewLogger("append"),
	}
	return f, nil
}

func (f *appendProcessor) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if f.config.FailOnError {
		backup = event.Clone()
	}

	err := f.appendValues(f.config.TargetField, f.config.Fields, f.config.Values, event)
	if err != nil {
		errMsg := fmt.Errorf("failed to append fields in append processor: %w", err)
		if management.TraceLevelEnabled() {
			f.logger.Debug(errMsg.Error())
		}
		if f.config.FailOnError {
			event = backup
			if _, err := event.PutValue("error.message", errMsg.Error()); err != nil {
				return nil, fmt.Errorf("failed to append fields in append processor: %w", err)
			}
			return event, err
		}
	}

	return event, nil
}

func (f *appendProcessor) appendValues(target string, fields []string, values []interface{}, event *beat.Event) error {
	var arr []interface{}

	// get the existing value of target field
	targetVal, err := event.GetValue(target)
	if err != nil {
		f.logger.Debugf("could not fetch value for key: '%s'. Therefore, all the values will be appended in a new key %s.", target, target)
	} else {
		targetArr, ok := targetVal.([]interface{})
		if ok {
			arr = append(arr, targetArr...)
		} else {
			arr = append(arr, targetVal)
		}
	}

	// append the values of all the fields listed under 'fields' section
	for _, field := range fields {
		val, err := event.GetValue(field)
		if err != nil {
			if f.config.IgnoreMissing && err.Error() == "key not found" {
				continue
			}
			return fmt.Errorf("could not fetch value for key: %s, Error: %w", field, err)
		}
		valArr, ok := val.([]interface{})
		if ok {
			arr = append(arr, valArr...)
		} else {
			arr = append(arr, val)
		}
	}

	// append all the static values from 'values' section
	arr = append(arr, values...)

	// remove empty strings and nil from the array
	if f.config.IgnoreEmptyValues {
		arr = cleanEmptyValues(arr)
	}

	// remove duplicate values from the array
	if !f.config.AllowDuplicate {
		arr = removeDuplicates(arr)
	}

	// replace the existing target with new array
	if err := event.Delete(target); err != nil && !(err.Error() == "key not found") {
		return fmt.Errorf("unable to delete the target field %s due to error: %w", target, err)
	}
	if _, err := event.PutValue(target, arr); err != nil {
		return fmt.Errorf("unable to put values in the target field %s due to error: %w", target, err)
	}

	return nil
}

func (f *appendProcessor) String() string {
	return "append=" + fmt.Sprintf("%+v", f.config.TargetField)
}

// this function will remove all the empty strings and nil values from the array
func cleanEmptyValues(dirtyArr []interface{}) (cleanArr []interface{}) {
	for _, val := range dirtyArr {
		if val == "" || val == nil {
			continue
		}
		cleanArr = append(cleanArr, val)
	}
	return cleanArr
}

// this function will remove all the duplicate values from the array
func removeDuplicates(dirtyArr []interface{}) (cleanArr []interface{}) {
	set := make(map[interface{}]bool, 0)
	for _, val := range dirtyArr {
		if _, ok := set[val]; !ok {
			set[val] = true
			cleanArr = append(cleanArr, val)
		}
	}
	return cleanArr
}
