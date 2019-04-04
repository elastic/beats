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
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type decodeCSVField struct {
	csvConfig
	separator rune
}

type csvConfig struct {
	Field            string `config:"field"`
	Target           string `config:"target"`
	IgnoreMissing    bool   `config:"ignore_missing"`
	TrimLeadingSpace bool   `config:"trim_leading_space"`
	OverwriteKeys    bool   `config:"overwrite_keys"`
	Separator        string `config:"separator"`
}

var (
	defaultCSVConfig = csvConfig{
		Separator: ",",
		Target:    "csv",
	}

	errFieldAlreadySet = errors.New("field already has a value")
)

func init() {
	processors.RegisterPlugin("decode_csv_field",
		configChecked(NewDecodeCSVField,
			requireFields("field"),
			allowedFields("field", "target", "ignore_missing", "overwrite_keys", "separator", "trim_leading_space", "overwrite_keys")))
}

// NewDecodeCSVField construct a new decode_csv_field processor.
func NewDecodeCSVField(c *common.Config) (processors.Processor, error) {
	config := defaultCSVConfig

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the decode_csv_field configuration: %s", err)
	}

	f := &decodeCSVField{csvConfig: config}
	switch runes := []rune(config.Separator); len(runes) {
	case 0:
		break
	case 1:
		f.separator = runes[0]
	default:
		return nil, errors.Errorf("separator must be a single character, got %d in string '%s'", len(runes), config.Separator)
	}
	return f, nil
}

// Run applies the decode_csv_field processor to an event.
func (f *decodeCSVField) Run(event *beat.Event) (*beat.Event, error) {
	data, err := event.GetValue(f.Field)
	if err != nil {
		if f.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return event, nil
		}
		return event, errors.Wrapf(err, "could not fetch value for field %s", f.Field)
	}

	text, ok := data.(string)
	if !ok {
		return event, errors.Errorf("field %s is not of string type", f.Field)
	}

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = f.separator
	reader.TrimLeadingSpace = f.TrimLeadingSpace
	// LazyQuotes makes the parser more tolerant to bad string formatting.
	reader.LazyQuotes = true

	record, err := reader.Read()
	if err != nil {
		return event, errors.Wrapf(err, "error decoding CSV from field %s", f.Field)
	}

	if !f.OverwriteKeys {
		if _, err = event.GetValue(f.Target); err == nil {
			return event, errors.Errorf("target field %s already has a value. Set the overwrite_keys flag or drop/rename the field first", f.Target)
		}
	}
	if _, err = event.PutValue(f.Target, record); err != nil {
		return event, errors.Wrapf(err, "failed setting field %s", f.Target)
	}
	return event, nil
}

func (f decodeCSVField) String() string {
	return "decode_csv_field=" + f.Field
}
