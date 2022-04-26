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

package extract_array

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type config struct {
	Field         string   `config:"field"`
	Mappings      mapstr.M `config:"mappings"`
	IgnoreMissing bool     `config:"ignore_missing"`
	OmitEmpty     bool     `config:"omit_empty"`
	OverwriteKeys bool     `config:"overwrite_keys"`
	FailOnError   bool     `config:"fail_on_error"`
}

type fieldMapping struct {
	from int
	to   string
}

type extractArrayProcessor struct {
	config
	mappings []fieldMapping
}

var (
	defaultConfig = config{
		FailOnError: true,
	}
	errNoMappings = errors.New("no mappings defined in extract_array processor")
)

func init() {
	processors.RegisterPlugin("extract_array",
		checks.ConfigChecked(New,
			checks.RequireFields("field", "mappings"),
			checks.AllowedFields("field", "mappings", "ignore_missing", "overwrite_keys", "fail_on_error", "when", "omit_empty")))

	jsprocessor.RegisterPlugin("ExtractArray", New)
}

// Unpack unpacks the processor's configuration.
func (f *extractArrayProcessor) Unpack(from *common.Config) error {
	tmp := defaultConfig
	err := from.Unpack(&tmp)
	if err != nil {
		return fmt.Errorf("failed to unpack the extract_array configuration: %s", err)
	}
	f.config = tmp
	for field, column := range f.Mappings.Flatten() {
		colIdx, ok := common.TryToInt(column)
		if !ok || colIdx < 0 {
			return fmt.Errorf("bad extract_array mapping for field %s: %+v is not a positive integer", field, column)
		}
		f.mappings = append(f.mappings, fieldMapping{from: colIdx, to: field})
	}
	sort.Slice(f.mappings, func(i, j int) bool {
		return f.mappings[i].from < f.mappings[j].from
	})
	return nil
}

// New builds a new extract_array processor.
func New(c *common.Config) (processors.Processor, error) {
	p := &extractArrayProcessor{}
	err := c.Unpack(p)
	if err != nil {
		return nil, err
	}
	if len(p.mappings) == 0 {
		return nil, errNoMappings
	}
	return p, nil
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Slice, reflect.Map:
		return v.IsNil() || v.Len() == 0
	case reflect.Interface:
		return v.IsNil() || isEmpty(v.Elem())
	}
	return false
}

func (f *extractArrayProcessor) Run(event *beat.Event) (*beat.Event, error) {
	iValue, err := event.GetValue(f.config.Field)
	if err != nil {
		if f.config.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return event, nil
		}
		return event, errors.Wrapf(err, "could not fetch value for field %s", f.config.Field)
	}

	array := reflect.ValueOf(iValue)
	if t := array.Type(); t.Kind() != reflect.Slice {
		if !f.config.FailOnError {
			return event, nil
		}
		return event, errors.Wrapf(err, "unsupported type for field %s: got: %s needed: array", f.config.Field, t.String())
	}

	saved := event
	if f.config.FailOnError {
		saved = event.Clone()
	}

	n := array.Len()
	for _, mapping := range f.mappings {
		if mapping.from >= n {
			if !f.config.FailOnError {
				continue
			}
			return saved, errors.Errorf("index %d exceeds length of %d when processing mapping for field %s", mapping.from, n, mapping.to)
		}
		cell := array.Index(mapping.from)
		// checking for CanInterface() here is done to prevent .Interface() from
		// panicking, but it can only happen when value points to a private
		// field inside a struct.
		if !cell.IsValid() || !cell.CanInterface() || (f.config.OmitEmpty && isEmpty(cell)) {
			continue
		}
		if !f.config.OverwriteKeys {
			if _, err = event.GetValue(mapping.to); err == nil {
				if !f.config.FailOnError {
					continue
				}
				return saved, errors.Errorf("target field %s already has a value. Set the overwrite_keys flag or drop/rename the field first", mapping.to)
			}
		}
		if _, err = event.PutValue(mapping.to, clone(cell.Interface())); err != nil {
			if !f.config.FailOnError {
				continue
			}
			return saved, errors.Wrapf(err, "failed setting field %s", mapping.to)
		}
	}
	return event, nil
}

func (f *extractArrayProcessor) String() (r string) {
	return fmt.Sprintf("extract_array={field=%s, mappings=%v}", f.config.Field, f.mappings)
}

func clone(value interface{}) interface{} {
	// TODO: This is dangerous but done by most processors.
	//       Otherwise need to reflect value and deep copy lists / map types.
	switch v := value.(type) {
	case mapstr.M:
		return v.Clone()
	case map[string]interface{}:
		return mapstr.M(v).Clone()
	case []interface{}:
		len := len(v)
		newArr := make([]interface{}, len)
		for idx, val := range v {
			newArr[idx] = clone(val)
		}
		return newArr
	}
	return value
}
