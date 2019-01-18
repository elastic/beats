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
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"

	"github.com/mitchellh/hashstructure"
)

var (
	moduleList = monitoring.NewUniqueList()
)

func init() {
	monitoring.NewFunc(monitoring.GetNamespace("state").GetRegistry(), "module", moduleList.Report, monitoring.Report)
}

// Factory for modules
type Factory struct {
	outlet                channel.Factory
	registrar             *registrar.Registrar
	beatVersion           string
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
	pipelineCallbackID    uuid.UUID
	beatDone              chan struct{}
}

// Wrap an array of inputs and implements cfgfile.Runner interface
type inputsRunner struct {
	id                    uint64
	moduleRegistry        *ModuleRegistry
	inputs                []*input.Runner
	pipelineLoaderFactory PipelineLoaderFactory
	pipelineCallbackID    uuid.UUID
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
		pipelineCallbackID:    uuid.Nil,
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
		pipelineCallbackID:    f.pipelineCallbackID,
		overwritePipelines:    f.overwritePipelines,
	}, nil
}

// CheckConfig checks if a config is valid or not
func (f *Factory) CheckConfig(config *common.Config) error {
	// TODO: add code here once we know that spinning up a filebeat input to check for errors doesn't cause memory leaks.
	return nil
}

func (p *inputsRunner) Start() {
	// Load pipelines
	if p.pipelineLoaderFactory != nil {
		// Attempt to load pipelines instantly when starting or after reload.
		// Thus, if ES was not available previously, it could be loaded this time.
		// If the function below fails, it means that ES is not available
		// at the moment, so the pipeline loader cannot be created.
		// Registering a callback regardless of the availability of ES
		// makes it possible to try to load pipeline when ES becomes reachable.
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

		// Register callback to try to load pipelines when connecting to ES.
		callback := func(esClient *elasticsearch.Client) error {
			return p.moduleRegistry.LoadPipelines(esClient, p.overwritePipelines)
		}
		p.pipelineCallbackID, err = elasticsearch.RegisterConnectCallback(callback)
		if err != nil {
			logp.Err("Error registering connect callback for Elasticsearch to load pipelines: %v", err)
		}
	}

	for _, input := range p.inputs {
		input.Start()
	}

	// Loop through and add modules, only 1 normally
	for m := range p.moduleRegistry.registry {
		moduleList.Add(m)
	}
}
func (p *inputsRunner) Stop() {
	if p.pipelineCallbackID != uuid.Nil {
		elasticsearch.DeregisterConnectCallback(p.pipelineCallbackID)
	}

	for _, input := range p.inputs {
		input.Stop()
	}

	// Loop through and remove modules, only 1 normally
	for m := range p.moduleRegistry.registry {
		moduleList.Remove(m)
	}
}

func (p *inputsRunner) String() string {
	return p.moduleRegistry.InfoString()
}
