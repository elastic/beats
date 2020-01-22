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

package input

import (
	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

// RunnerFactory is a factory for registrars
type RunnerFactory struct {
	outlet    channel.Factory
	registrar *registrar.Registrar
	beatDone  chan struct{}
}

// NewRunnerFactory instantiates a new RunnerFactory
func NewRunnerFactory(outlet channel.Factory, registrar *registrar.Registrar, beatDone chan struct{}) *RunnerFactory {
	return &RunnerFactory{
		outlet:    outlet,
		registrar: registrar,
		beatDone:  beatDone,
	}
}

// Create creates a input based on a config
func (r *RunnerFactory) Create(
	pipeline beat.Pipeline,
	c *common.Config,
	meta *common.MapStrPointer,
) (cfgfile.Runner, error) {
	connector := r.outlet(pipeline)
	p, err := New(c, connector, r.beatDone, r.registrar.GetStates(), meta)
	if err != nil {
		// In case of error with loading state, input is still returned
		return p, err
	}

	return p, nil
}
