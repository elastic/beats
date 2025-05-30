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

package grok

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	"github.com/elastic/elastic-agent-libs/config"
	gogrok "github.com/elastic/go-grok"
)

func init() {
	processors.RegisterPlugin("grok",
		checks.ConfigChecked(NewGrok, checks.RequireFields("field", "pattern")))
}

type grokConfig struct {
	Pattern        string            `config:"pattern"`
	Field          string            `config:"field"`
	CustomPatterns map[string]string `config:"customPatterns"`
}

type grokProcessor struct {
	config grokConfig
	grok   gogrok.Grok
}

func (u grokProcessor) Run(event *beat.Event) (*beat.Event, error) {
	root := event.Fields.Clone()
	field, err := root.GetValue(u.config.Field)
	if err != nil {
		return nil, fmt.Errorf("failed to get field '%s' from event: %w", field, err)
	}

	input, ok := field.(string)
	if !ok {
		return nil, fmt.Errorf("field '%s' is not a string", field)
	}

	values, err := u.grok.ParseTypedString(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input with grok pattern: %w", err)
	}

	for k, v := range values {
		_, err := event.PutValue(k, v)
		if err != nil {
			return nil, fmt.Errorf("failed to update event with parsed data: %w", err)
		}
	}

	return event, nil
}

func (u grokProcessor) String() string {
	return "grok"
}

func NewGrok(c *config.C) (beat.Processor, error) {
	gc := grokConfig{}
	err := c.Unpack(&gc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack grok config: %w", err)
	}

	gk, err := gogrok.NewComplete(gc.CustomPatterns)
	if err != nil {
		return nil, fmt.Errorf("faild to build grok parser: %w", err)
	}

	err = gk.Compile(gc.Pattern, true)
	if err != nil {
		return nil, fmt.Errorf("cannot compile pattern %w", gc.Pattern, err)
	}

	p := &grokProcessor{
		config: gc,
		grok:   *gk,
	}

	return p, nil
}
