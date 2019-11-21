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

package v2

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Loader interface {
	Configure(log *logp.Logger, config *common.Config) (Input, error)
}

type InputLoader struct {
	plugins map[string]Plugin

	DefaultType string
}

type inputTypeConfig struct {
	Type string `config:"type"`
}

func NewInputLoader(
	plugins ...Plugin,
) (*InputLoader, error) {
	m := make(map[string]Plugin, len(plugins))
	for _, p := range plugins {
		name := p.Name
		if _, exists := m[name]; exists {
			return nil, fmt.Errorf("duplicate plugin '%v'", name)
		}

		m[name] = p
	}

	return &InputLoader{plugins: m}, nil
}

func (l *InputLoader) Configure(log *logp.Logger, config *common.Config) (Input, error) {
	typeConfig, err := unpackTypeConfig(l.DefaultType, config)
	if err != nil {
		return nil, err
	}

	plugin, exists := l.plugins[typeConfig.Type]
	if !exists {
		return nil, fmt.Errorf("unknown input type %v", typeConfig.Type)
	}

	return plugin.Configure(log, config)
}

func unpackTypeConfig(defaultType string, config *common.Config) (c inputTypeConfig, err error) {
	c = inputTypeConfig{Type: defaultType}
	err = config.Unpack(&c)
	if err != nil {
		return c, err
	}
	if c.Type == "" {
		return c, ErrNoTypeConfigured
	}
	return c, nil
}
