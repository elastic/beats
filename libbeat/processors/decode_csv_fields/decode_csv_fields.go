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

package decode_csv_fields

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/config"
)

type decodeCSVFields struct {
	csvConfig
	fields    map[string]string
	separator rune
}

type csvConfig struct {
	Fields           common.MapStr `config:"fields"`
	IgnoreMissing    bool          `config:"ignore_missing"`
	TrimLeadingSpace bool          `config:"trim_leading_space"`
	OverwriteKeys    bool          `config:"overwrite_keys"`
	FailOnError      bool          `config:"fail_on_error"`
	Separator        string        `config:"separator"`
}

var (
	defaultCSVConfig = csvConfig{
		Separator:   ",",
		FailOnError: true,
	}

	errFieldAlreadySet = errors.New("field already has a value")
)

func init() {
	processors.RegisterPlugin("decode_csv_fields",
		checks.ConfigChecked(NewDecodeCSVField,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "overwrite_keys", "separator", "trim_leading_space", "overwrite_keys", "fail_on_error", "when")))

	jsprocessor.RegisterPlugin("DecodeCSVField", NewDecodeCSVField)
}

// NewDecodeCSVField construct a new decode_csv_field processor.
func NewDecodeCSVField(c *config.C) (processors.Processor, error) {
	config := defaultCSVConfig

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the decode_csv_field configuration: %s", err)
	}
	if len(config.Fields) == 0 {
		return nil, errors.New("no fields to decode configured")
	}
	f := &decodeCSVFields{csvConfig: config}
	// Set separator as rune
	switch runes := []rune(config.Separator); len(runes) {
	case 0:
		break
	case 1:
		f.separator = runes[0]
	default:
		return nil, errors.Errorf("separator must be a single character, got %d in string '%s'", len(runes), config.Separator)
	}
	// Set fields as string -> string
	f.fields = make(map[string]string, len(config.Fields))
	for src, dstIf := range config.Fields.Flatten() {
		dst, ok := dstIf.(string)
		if !ok {
			return nil, errors.Errorf("bad destination mapping for %s: destination field must be string, not %T (got %v)", src, dstIf, dstIf)
		}
		f.fields[src] = dst
	}
	return f, nil
}

// Run applies the decode_csv_field processor to an event.
func (f *decodeCSVFields) Run(event *beat.Event) (*beat.Event, error) {
	var saved *beat.Event
	if f.FailOnError {
		saved = event.Clone()
	}
	for src, dest := range f.fields {
		if err := f.decodeCSVField(src, dest, event); err != nil && f.FailOnError {
			return saved, err
		}
	}
	return event, nil
}

func (f *decodeCSVFields) decodeCSVField(src, dest string, event *beat.Event) error {
	data, err := event.GetValue(src)
	if err != nil {
		if f.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return errors.Wrapf(err, "could not fetch value for field %s", src)
	}

	text, ok := data.(string)
	if !ok {
		return errors.Errorf("field %s is not of string type", src)
	}

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = f.separator
	reader.TrimLeadingSpace = f.TrimLeadingSpace
	// LazyQuotes makes the parser more tolerant to bad string formatting.
	reader.LazyQuotes = true

	record, err := reader.Read()
	if err != nil {
		return errors.Wrapf(err, "error decoding CSV from field %s", src)
	}

	if src != dest && !f.OverwriteKeys {
		if _, err = event.GetValue(dest); err == nil {
			return errors.Errorf("target field %s already has a value. Set the overwrite_keys flag or drop/rename the field first", dest)
		}
	}
	if _, err = event.PutValue(dest, record); err != nil {
		return errors.Wrapf(err, "failed setting field %s", dest)
	}
	return nil
}

// String returns a string representation of this processor.
func (f decodeCSVFields) String() string {
	json, _ := json.Marshal(f.csvConfig)
	return "decode_csv_field=" + string(json)
}
