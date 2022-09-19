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
		checks.ConfigChecked(NewMoveFields, checks.AllowedFields(
			"parent_path", "from", "to", "exclude", "ignore_from_not_found",
		)))
	jsprocessor.RegisterPlugin("MoveFields", NewMoveFields)
}

type moveFieldsConfig struct {
	ParentPath         string   `config:"parent_path"`
	From               []string `config:"from"`
	IgnoreFromNotFound bool     `config:"ignore_from_not_found"`
	To                 string   `config:"to"`
	Exclude            []string `config:"exclude"`

	excludeMap map[string]bool
}

type moveFields struct {
	config moveFieldsConfig
}

func (u moveFields) Run(event *beat.Event) (*beat.Event, error) {
	root := event.Fields.Clone()
	parent := root
	if p := u.config.ParentPath; p != "" {
		parentValue, err := root.GetValue(p)
		if err != nil {
			return nil, fmt.Errorf("move field read parent path field failed: %w", err)
		}
		var ok bool
		parent, ok = parentValue.(mapstr.M)
		if !ok {
			return nil, fmt.Errorf("move field parent is not message map")
		}
	}

	keys := u.config.From
	if len(keys) == 0 {
		keys = make([]string, 0, len(parent))
		for k := range parent {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		if _, ok := u.config.excludeMap[k]; ok {
			continue
		}
		v, err := parent.GetValue(k)
		if u.config.IgnoreFromNotFound && errors.Is(err, mapstr.ErrKeyNotFound) {
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

func NewMoveFields(c *config.C) (processors.Processor, error) {
	fc := moveFieldsConfig{}
	err := c.Unpack(&fc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack move fields config: %w", err)
	}

	fc.excludeMap = make(map[string]bool)
	for _, k := range fc.Exclude {
		fc.excludeMap[k] = true
	}

	return &moveFields{
		config: fc,
	}, nil
}
