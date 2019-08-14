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

package autodiscover

import (
	"errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

// AutodiscoverAdapter for Filebeat modules & input
type AutodiscoverAdapter struct {
	inputFactory  cfgfile.RunnerFactory
	moduleFactory cfgfile.RunnerFactory
}

// NewAutodiscoverAdapter builds and returns an autodiscover adapter for Filebeat modules & input
func NewAutodiscoverAdapter(inputFactory, moduleFactory cfgfile.RunnerFactory) *AutodiscoverAdapter {
	return &AutodiscoverAdapter{
		inputFactory:  inputFactory,
		moduleFactory: moduleFactory,
	}
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *AutodiscoverAdapter) CreateConfig(e bus.Event) ([]*common.Config, error) {
	config, ok := e["config"].([]*common.Config)
	if !ok {
		return nil, errors.New("Got a wrong value in event `config` key")
	}
	return config, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *AutodiscoverAdapter) CheckConfig(c *common.Config) error {
	var factory cfgfile.RunnerFactory

	if c.HasField("module") {
		factory = m.moduleFactory
	} else {
		factory = m.inputFactory
	}

	if checker, ok := factory.(cfgfile.ConfigChecker); ok {
		return checker.CheckConfig(c)
	}

	return nil
}

// Create a module or input from the given config
func (m *AutodiscoverAdapter) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	if c.HasField("module") {
		return m.moduleFactory.Create(p, c, meta)
	}
	return m.inputFactory.Create(p, c, meta)
}

// EventFilter returns the bus filter to retrieve runner start/stop triggering events
func (m *AutodiscoverAdapter) EventFilter() []string {
	return []string{"config"}
}
