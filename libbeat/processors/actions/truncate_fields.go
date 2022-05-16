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
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type truncateFieldsConfig struct {
	Fields        []string `config:"fields"`
	MaxBytes      int      `config:"max_bytes" validate:"min=0"`
	MaxChars      int      `config:"max_characters" validate:"min=0"`
	IgnoreMissing bool     `config:"ignore_missing"`
	FailOnError   bool     `config:"fail_on_error"`
}

type truncateFields struct {
	config   truncateFieldsConfig
	truncate truncater
	logger   *logp.Logger
}

type truncater func(*truncateFields, []byte) ([]byte, bool, error)

func init() {
	processors.RegisterPlugin("truncate_fields",
		checks.ConfigChecked(NewTruncateFields,
			checks.RequireFields("fields"),
			checks.MutuallyExclusiveRequiredFields("max_bytes", "max_characters"),
		),
	)
	jsprocessor.RegisterPlugin("TruncateFields", NewTruncateFields)
}

// NewTruncateFields returns a new truncate_fields processor.
func NewTruncateFields(c *conf.C) (processors.Processor, error) {
	var config truncateFieldsConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the truncate_fields configuration: %s", err)
	}

	var truncateFunc truncater
	if config.MaxBytes > 0 {
		truncateFunc = (*truncateFields).truncateBytes
	} else {
		truncateFunc = (*truncateFields).truncateCharacters
	}

	return &truncateFields{
		config:   config,
		truncate: truncateFunc,
		logger:   logp.NewLogger("truncate_fields"),
	}, nil
}

func (f *truncateFields) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if f.config.FailOnError {
		backup = event.Clone()
	}

	for _, field := range f.config.Fields {
		event, err := f.truncateSingleField(field, event)
		if err != nil {
			f.logger.Debugf("Failed to truncate fields: %s", err)
			if f.config.FailOnError {
				event = backup
				return event, err
			}
		}
	}

	return event, nil
}

func (f *truncateFields) truncateSingleField(field string, event *beat.Event) (*beat.Event, error) {
	v, err := event.GetValue(field)
	if err != nil {
		if f.config.IgnoreMissing && errors.Cause(err) == mapstr.ErrKeyNotFound {
			return event, nil
		}
		return event, errors.Wrapf(err, "could not fetch value for key: %s", field)
	}

	switch value := v.(type) {
	case []byte:
		return f.addTruncatedByte(field, value, event)
	case string:
		return f.addTruncatedString(field, value, event)
	default:
		return event, fmt.Errorf("value cannot be truncated: %+v", value)
	}

}

func (f *truncateFields) addTruncatedString(field, value string, event *beat.Event) (*beat.Event, error) {
	truncated, isTruncated, err := f.truncate(f, []byte(value))
	if err != nil {
		return event, err
	}
	_, err = event.PutValue(field, string(truncated))
	if err != nil {
		return event, fmt.Errorf("could not add truncated string value for key: %s, Error: %+v", field, err)
	}

	if isTruncated {
		mapstr.AddTagsWithKey(event.Fields, "log.flags", []string{"truncated"})
	}

	return event, nil
}

func (f *truncateFields) addTruncatedByte(field string, value []byte, event *beat.Event) (*beat.Event, error) {
	truncated, isTruncated, err := f.truncate(f, value)
	if err != nil {
		return event, err
	}
	_, err = event.PutValue(field, truncated)
	if err != nil {
		return event, fmt.Errorf("could not add truncated byte slice value for key: %s, Error: %+v", field, err)
	}

	if isTruncated {
		mapstr.AddTagsWithKey(event.Fields, "log.flags", []string{"truncated"})
	}

	return event, nil
}

func (f *truncateFields) truncateBytes(value []byte) ([]byte, bool, error) {
	size := len(value)
	if size <= f.config.MaxBytes {
		return value, false, nil
	}

	size = f.config.MaxBytes
	truncated := make([]byte, size)
	n := copy(truncated, value[:size])
	if n != size {
		return nil, false, fmt.Errorf("unexpected number of bytes were copied")
	}
	return truncated, true, nil
}

func (f *truncateFields) truncateCharacters(value []byte) ([]byte, bool, error) {
	count := utf8.RuneCount(value)
	if count <= f.config.MaxChars {
		return value, false, nil
	}

	count = f.config.MaxChars
	r := bytes.NewReader(value)
	w := bytes.NewBuffer(nil)

	for i := 0; i < count; i++ {
		r, _, err := r.ReadRune()
		if err != nil {
			return nil, false, err
		}

		_, err = w.WriteRune(r)
		if err != nil {
			return nil, false, err
		}
	}

	return w.Bytes(), true, nil
}

func (f *truncateFields) String() string {
	var limit string
	if f.config.MaxBytes > 0 {
		limit = fmt.Sprintf("max_bytes=%d", f.config.MaxBytes)
	} else {
		limit = fmt.Sprintf("max_characters=%d", f.config.MaxChars)
	}
	return "truncate_fields=" + strings.Join(f.config.Fields, ", ") + limit
}
