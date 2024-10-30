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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

// alterFieldFunc defines how fields must be processed
type alterFieldFunc func(field string) string

type alterFieldProcessor struct {
	Fields        []string
	IgnoreMissing bool
	FailOnError   bool
	FullPath      bool

	processorName string
	alterFunc     alterFieldFunc
}

// NewAlterFieldProcessor is an umbrella method for processing events based on provided fields. Such as converting event keys to uppercase/lowercase
func NewAlterFieldProcessor(c *conf.C, processorName string, alterFunc alterFieldFunc) (beat.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		FailOnError   bool     `config:"fail_on_error"`
		FullPath      bool     `config:"full_path"`
	}{
		IgnoreMissing: false,
		FailOnError:   true,
		FullPath:      true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the %s fields configuration: %s", processorName, err)
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
		Fields:        config.Fields,
		IgnoreMissing: config.IgnoreMissing,
		FailOnError:   config.FailOnError,
		processorName: processorName,
		FullPath:      config.FullPath,
		alterFunc:     alterFunc,
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
			if a.FailOnError {
				event = backup
				event.PutValue("error.message", err.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (a *alterFieldProcessor) alter(event *beat.Event, field string) error {

	var key string
	var value interface{}
	var err error

	// Get the value matching the key
	// searches full path 'case insensitively'
	if a.FullPath || (!a.FullPath && !strings.ContainsRune(field, '.')) {
		key, value, err = event.Fields.FindFold(field)
	} else {
		// searches for only the most nested key 'case insensitively'
		idx := lastIndexDot(field)
		value, err = event.Fields.GetValue(field[:idx])
		if err != nil {
			if a.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
				return nil
			}
			return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, err)
		}

		current, mapType := tryToMapStr(value)
		if !mapType {
			return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, mapstr.ErrKeyNotFound)

		}
		key, value, err = current.FindFold(field[idx+1:])
		key = field[:idx+1] + key
	}

	// If err is not nil for any of the above case
	if err != nil {
		if a.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, err)
	}

	// Delete the existing value
	if err := event.Delete(key); err != nil {
		return fmt.Errorf("could not delete field: %s, Error: %v", key, err)
	}

	// Alter the field
	var alterString string
	if strings.ContainsRune(key, '.') {
		// In case of nested fields provided, we need to make sure to only modify the last field segment in the chain
		idx := lastIndexDot(key)
		alterString = key[:idx+1] + a.alterFunc(key[idx+1:])
	} else {
		alterString = a.alterFunc(key)
	}

	// Put the value back
	if _, err := event.PutValue(alterString, value); err != nil {
		return fmt.Errorf("could not put value: %s: %v, Error: %v", alterString, value, err)
	}

	return nil
}

func lastIndexDot(key string) (idx int) {
	lastIndexRuneFunc := func(r rune) bool { return r == '.' }
	idx = strings.LastIndexFunc(key, lastIndexRuneFunc)
	return idx
}

func toMapStr(v interface{}) (mapstr.M, error) {
	m, ok := tryToMapStr(v)
	if !ok {
		return nil, fmt.Errorf("expected map but type is %T", v)
	}
	return m, nil
}

func tryToMapStr(v interface{}) (mapstr.M, bool) {
	switch m := v.(type) {
	case mapstr.M:
		return m, true
	case map[string]interface{}:
		return mapstr.M(m), true
	default:
		return nil, false
	}
}
