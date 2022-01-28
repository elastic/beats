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
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
)

var (
	moduleList = monitoring.NewUniqueList()
)

func init() {
	monitoring.NewFunc(monitoring.GetNamespace("state").GetRegistry(), "module", moduleList.Report, monitoring.Report)
}

// Factory for modules
type Factory struct {
	beatInfo              beat.Info
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
	pipelineCallbackID    uuid.UUID
	inputFactory          cfgfile.RunnerFactory
}

// Wrap an array of inputs and implements cfgfile.Runner interface
type inputsRunner struct {
	id                    uint64
	moduleRegistry        *ModuleRegistry
	inputs                []cfgfile.Runner
	pipelineLoaderFactory PipelineLoaderFactory
	pipelineCallbackID    uuid.UUID
	overwritePipelines    bool
	log                   *logp.Logger
}

// NewFactory instantiates a new Factory
func NewFactory(
	inputFactory cfgfile.RunnerFactory,
	beatInfo beat.Info,
	pipelineLoaderFactory PipelineLoaderFactory,
	overwritePipelines bool,
) *Factory {
	return &Factory{
		inputFactory:          inputFactory,
		beatInfo:              beatInfo,
		pipelineLoaderFactory: pipelineLoaderFactory,
		pipelineCallbackID:    uuid.Nil,
		overwritePipelines:    overwritePipelines,
	}
}

// Create creates a module based on a config
func (f *Factory) Create(p beat.PipelineConnector, c *common.Config) (cfgfile.Runner, error) {
	m, pConfigs, err := f.createRegistry(c)
	if err != nil {
		return nil, fmt.Errorf("could not create module registry for filesets: %w", err)
	}

	// Hash module ID
	var h map[string]interface{}
	if err = c.Unpack(&h); err != nil {
		return nil, fmt.Errorf("failed to unpack config: %w", err)
	}
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	inputs := make([]cfgfile.Runner, len(pConfigs))
	for i, pConfig := range pConfigs {
		inputs[i], err = f.inputFactory.Create(p, pConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create input: %w", err)
		}
	}

	return &inputsRunner{
		id:                    id,
		moduleRegistry:        m,
		inputs:                inputs,
		pipelineLoaderFactory: f.pipelineLoaderFactory,
		pipelineCallbackID:    f.pipelineCallbackID,
		overwritePipelines:    f.overwritePipelines,
		log:                   logp.NewLogger(logName),
	}, nil
}

func (f *Factory) CheckConfig(c *common.Config) error {
	_, pConfigs, err := f.createRegistry(c)
	if err != nil {
		return fmt.Errorf("could not create module registry for filesets: %w", err)
	}

	for _, pConfig := range pConfigs {
		err = f.inputFactory.CheckConfig(pConfig)
		if err != nil {
			return fmt.Errorf("error checking input configuration: %w", err)
		}
	}

	return nil
}

// createRegistry starts a registry for a set of filesets, it returns the registry and
// its input configurations
func (f *Factory) createRegistry(c *common.Config) (*ModuleRegistry, []*common.Config, error) {
	m, err := NewModuleRegistry([]*common.Config{c}, f.beatInfo, false)
	if err != nil {
		return nil, nil, err
	}

	pConfigs, err := m.GetInputConfigs()
	if err != nil {
		return nil, nil, err
	}

	return m, pConfigs, err
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
			p.log.Errorf("Error loading pipeline: %s", err)
		} else {
			err := p.moduleRegistry.LoadPipelines(pipelineLoader, p.overwritePipelines)
			if err != nil {
				// Log error and continue
				p.log.Errorf("Error loading pipeline: %s", err)
			}
		}

		// Register callback to try to load pipelines when connecting to ES.
		callback := func(esClient *eslegclient.Connection) error {
			return p.moduleRegistry.LoadPipelines(esClient, p.overwritePipelines)
		}
		p.pipelineCallbackID, err = elasticsearch.RegisterConnectCallback(callback)
		if err != nil {
			p.log.Errorf("Error registering connect callback for Elasticsearch to load pipelines: %v", err)
		}
	}

	for _, input := range p.inputs {
		input.Start()
	}

	// Loop through and add modules
	for _, modules := range p.moduleRegistry.registry {
		for m := range modules {
			moduleList.Add(m)
		}
	}
}

func (p *inputsRunner) Stop() {
	if p.pipelineCallbackID != uuid.Nil {
		elasticsearch.DeregisterConnectCallback(p.pipelineCallbackID)
	}

	for _, input := range p.inputs {
		input.Stop()
	}

	// Loop through and remove modules
	for _, modules := range p.moduleRegistry.registry {
		for m := range modules {
			moduleList.Remove(m)
		}
	}
}

func (p *inputsRunner) String() string {
	return p.moduleRegistry.InfoString()
}
