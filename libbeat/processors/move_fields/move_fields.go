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

package move_fields

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	processors.RegisterPlugin("move_fields",
		checks.ConfigChecked(NewMoveFields, checks.RequireFields("to")))
	jsprocessor.RegisterPlugin("MoveFields", NewMoveFields)
}

type moveFieldsConfig struct {
	Exclude       []string `config:"exclude"`
	Fields        []string `config:"fields"`
	From          string   `config:"from"`
	To            string   `config:"to"`
	IgnoreMissing bool     `config:"ignore_missing"`
}

type moveFields struct {
	config     moveFieldsConfig
	excludeMap map[string]struct{}
}

func (u moveFields) Run(event *beat.Event) (*beat.Event, error) {
	root := event.Fields.Clone()
	parent := root
	if p := u.config.From; p != "" {
		parentValue, err := root.GetValue(p)
		if err != nil {
			return nil, fmt.Errorf("cannot get value from key '%s': %w", p, err)
		}
		var ok bool
		parent, ok = parentValue.(mapstr.M)
		if !ok {
			return nil, fmt.Errorf("'%s' does not contain an object, it contains '%#v'", p, parentValue)
		}
	}

	keys := u.config.Fields
	if len(keys) == 0 {
		keys = make([]string, 0, len(parent))
		for k := range parent {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		if _, ok := u.excludeMap[k]; ok {
			continue
		}
		v, err := parent.GetValue(k)
		if u.config.IgnoreMissing && errors.Is(err, mapstr.ErrKeyNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("move field read field from parent, sub key: %s, failed: %w", k, err)
		}
		if err = parent.Delete(k); err != nil {
			return nil, fmt.Errorf("move field delete field from parent sub key: %s, failed: %w", k, err)
		}
		newKey := fmt.Sprintf("%s%s", u.config.To, k)
		if _, err = root.Put(newKey, v); err != nil {
			return nil, fmt.Errorf("move field write field to sub key: %s, new key: %s, failed: %w", k, newKey, err)
		}
	}

	event.Fields = root
	return event, nil
}

func (u moveFields) String() string {
	return "move_fields"
}

func NewMoveFields(c *config.C) (beat.Processor, error) {
	fc := moveFieldsConfig{}
	err := c.Unpack(&fc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack move fields config: %w", err)
	}

	p := &moveFields{
		config:     fc,
		excludeMap: make(map[string]struct{}),
	}
	for _, k := range fc.Exclude {
		p.excludeMap[k] = struct{}{}
	}

	return p, nil
}
