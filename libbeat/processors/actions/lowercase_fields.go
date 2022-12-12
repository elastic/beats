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
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

type lowerCaseProcessor struct {
	Fields        []string
	IgnoreMissing bool
	FailOnError   bool
}

func init() {
	processors.RegisterPlugin(
		"lowercase_fields",
		checks.ConfigChecked(
			NewLowerCaseProcessor,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "when", "ignore_missing", "fail_on_error"),
		),
	)
}

func NewLowerCaseProcessor(c *conf.C) (processors.Processor, error) {
	config := struct {
		Fields        []string `config:"fields"`
		IgnoreMissing bool     `config:"ignore_missing"`
		FailOnError   bool     `config:"fail_on_error"`
	}{
		IgnoreMissing: false,
		FailOnError:   true,
	}

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("failed to unpack the lowercase_fields configuration: %s", err)
	}

	// Skip mandatory fields
	for _, readOnly := range processors.MandatoryExportedFields {
		for i, field := range config.Fields {
			if field == readOnly {
				config.Fields = append(config.Fields[:i], config.Fields[i+1:]...)
			}
		}
	}

	return &lowerCaseProcessor{Fields: config.Fields, IgnoreMissing: config.IgnoreMissing, FailOnError: config.FailOnError}, nil
}

func (p *lowerCaseProcessor) Run(event *beat.Event) (*beat.Event, error) {
	var backup *beat.Event
	if p.FailOnError {
		backup = event.Clone()
	}

	for _, field := range p.Fields {
		if err := p.lowerCaseField(event, field); err != nil {
			if p.FailOnError {
				event = backup
				event.PutValue("error.message", err.Error())
				return event, err
			}
		}
	}

	return event, nil
}

func (p *lowerCaseProcessor) lowerCaseField(event *beat.Event, field string) error {
	value, err := event.GetValue(field)
	if err != nil {
		if p.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %v", field, err)
	}

	if err := event.Delete(field); err != nil {
		return fmt.Errorf("could not delete key: %s, Error: %v", field, err)
	}

	var lower string
	if strings.ContainsRune(field, '.') {
		// In case of nested fields provided, we need to make sure to only modify the latest field in the chain
		lastIndexRuneFunc := func(r rune) bool { return r == '.' }
		idx := strings.LastIndexFunc(field, lastIndexRuneFunc)
		lower = field[:idx+1] + strings.ToLower(field[idx+1:])
	} else {
		lower = strings.ToLower(field)
	}

	if _, err := event.PutValue(lower, value); err != nil {
		return fmt.Errorf("could not put value: %s: %v, Error: %v", lower, value, err)
	}

	return nil
}

func (p *lowerCaseProcessor) String() string {
	return fmt.Sprintf("lowercase_fields=%+v", *p)
}
