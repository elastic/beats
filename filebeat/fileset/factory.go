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

package fileset

import (
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	input "github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

// Factory for modules
type Factory struct {
	outlet                channel.Factory
	registrar             *registrar.Registrar
	beatVersion           string
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
	beatDone              chan struct{}
}

// Wrap an array of inputs and implements cfgfile.Runner interface
type inputsRunner struct {
	id                    uint64
	moduleRegistry        *ModuleRegistry
	inputs                []*input.Runner
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
}

// NewFactory instantiates a new Factory
func NewFactory(outlet channel.Factory, registrar *registrar.Registrar, beatVersion string,
	pipelineLoaderFactory PipelineLoaderFactory, overwritePipelines bool, beatDone chan struct{}) *Factory {
	return &Factory{
		outlet:                outlet,
		registrar:             registrar,
		beatVersion:           beatVersion,
		beatDone:              beatDone,
		pipelineLoaderFactory: pipelineLoaderFactory,
		overwritePipelines:    overwritePipelines,
	}
}

// Create creates a module based on a config
func (f *Factory) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	// Start a registry of one module:
	m, err := NewModuleRegistry([]*common.Config{c}, f.beatVersion, false)
	if err != nil {
		return nil, err
	}

	pConfigs, err := m.GetInputConfigs()
	if err != nil {
		return nil, err
	}

	// Hash module ID
	var h map[string]interface{}
	c.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	inputs := make([]*input.Runner, len(pConfigs))
	connector := channel.ConnectTo(p, f.outlet)
	for i, pConfig := range pConfigs {
		inputs[i], err = input.New(pConfig, connector, f.beatDone, f.registrar.GetStates(), meta)
		if err != nil {
			logp.Err("Error creating input: %s", err)
			return nil, err
		}
	}

	return &inputsRunner{
		id:                    id,
		moduleRegistry:        m,
		inputs:                inputs,
		pipelineLoaderFactory: f.pipelineLoaderFactory,
		overwritePipelines:    f.overwritePipelines,
	}, nil
}

func (p *inputsRunner) Start() {
	// Load pipelines
	if p.pipelineLoaderFactory != nil {
		// Load pipelines instantly and then setup a callback for reconnections:
		pipelineLoader, err := p.pipelineLoaderFactory()
		if err != nil {
			logp.Err("Error loading pipeline: %s", err)
		} else {
			err := p.moduleRegistry.LoadPipelines(pipelineLoader, p.overwritePipelines)
			if err != nil {
				// Log error and continue
				logp.Err("Error loading pipeline: %s", err)
			}
		}

		// Callback:
		callback := func(esClient *elasticsearch.Client) error {
			return p.moduleRegistry.LoadPipelines(esClient, p.overwritePipelines)
		}
		elasticsearch.RegisterConnectCallback(callback)
	}

	for _, input := range p.inputs {
		input.Start()
	}
}
func (p *inputsRunner) Stop() {
	for _, input := range p.inputs {
		input.Stop()
	}
}

func (p *inputsRunner) String() string {
	return p.moduleRegistry.InfoString()
}
