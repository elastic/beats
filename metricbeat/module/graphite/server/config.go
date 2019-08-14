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

package server

import (
	"errors"
)

const (
	defaultDelimiter = "."
)

type GraphiteServerConfig struct {
	Protocol        string           `config:"protocol"`
	Templates       []TemplateConfig `config:"templates"`
	DefaultTemplate TemplateConfig   `config:"default_template"`
}

type TemplateConfig struct {
	Filter    string            `config:"filter"`
	Template  string            `config:"template"`
	Namespace string            `config:"namespace"`
	Delimiter string            `config:"delimiter"`
	Tags      map[string]string `config:"tags"`
}

func DefaultGraphiteCollectorConfig() GraphiteServerConfig {
	return GraphiteServerConfig{
		Protocol: "udp",
		DefaultTemplate: TemplateConfig{
			Filter:    "*",
			Template:  "metric*",
			Namespace: "graphite",
			Delimiter: ".",
		},
	}
}

func (c GraphiteServerConfig) Validate() error {
	if c.Protocol != "tcp" && c.Protocol != "udp" {
		return errors.New("`protocol` can only be tcp or udp")
	}
	return nil
}

func (t *TemplateConfig) Validate() error {
	if t.Namespace == "" {
		return errors.New("`namespace` can not be empty in template configuration")
	}

	if t.Filter == "" {
		return errors.New("`filter` can not be empty in template configuration")
	}

	if t.Template == "" {
		return errors.New("`template` can not be empty in template configuration")
	}

	if t.Delimiter == "" {
		t.Delimiter = defaultDelimiter
	}

	return nil
}
