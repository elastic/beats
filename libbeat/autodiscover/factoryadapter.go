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

// FactoryAdapter is an adapter that works with any cfgfile.RunnerFactory.
type FactoryAdapter struct {
	factory cfgfile.CheckableRunnerFactory
}

// NewFactoryAdapter builds and returns an autodiscover adapter that works with any cfgfile.RunnerFactory.
func NewFactoryAdapter(factory cfgfile.CheckableRunnerFactory) *FactoryAdapter {
	return &FactoryAdapter{
		factory: factory,
	}
}

// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
func (m *FactoryAdapter) CreateConfig(e bus.Event) ([]*common.Config, error) {
	config, ok := e["config"].([]*common.Config)
	if !ok {
		return nil, errors.New("Got a wrong value in event `config` key")
	}
	return config, nil
}

// CheckConfig tests given config to check if it will work or not, returns errors in case it won't work
func (m *FactoryAdapter) CheckConfig(c *common.Config) error {
	return m.factory.CheckConfig(c)
}

// Create a module or prospector from the given config
func (m *FactoryAdapter) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	return m.factory.Create(p, c, meta)
}

// EventFilter returns the bus filter to retrieve runner start/stop triggering events
func (m *FactoryAdapter) EventFilter() []string {
	return []string{"config"}
}
