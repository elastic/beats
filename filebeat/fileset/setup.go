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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	pubpipeline "github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// SetupFactory is for loading module assets when running setup subcommand.
type SetupFactory struct {
	beatInfo              beat.Info
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
}

// NewSetupFactory creates a SetupFactory
func NewSetupFactory(beatInfo beat.Info, pipelineLoaderFactory PipelineLoaderFactory) *SetupFactory {
	return &SetupFactory{
		beatInfo:              beatInfo,
		pipelineLoaderFactory: pipelineLoaderFactory,
		overwritePipelines:    true,
	}
}

// Create creates a new SetupCfgRunner to setup module configuration.
func (sf *SetupFactory) Create(_ beat.PipelineConnector, c *conf.C) (cfgfile.Runner, error) {
	m, err := NewModuleRegistry([]*conf.C{c}, sf.beatInfo, false)
	if err != nil {
		return nil, err
	}

	return &SetupCfgRunner{
		moduleRegistry:        m,
		pipelineLoaderFactory: sf.pipelineLoaderFactory,
		overwritePipelines:    sf.overwritePipelines,
	}, nil
}

func (sf *SetupFactory) CheckConfig(c *conf.C) error {
	_, err := sf.Create(pubpipeline.NewNilPipeline(), c)
	return err
}

// SetupCfgRunner is for loading assets of modules.
type SetupCfgRunner struct {
	moduleRegistry        *ModuleRegistry
	pipelineLoaderFactory PipelineLoaderFactory
	overwritePipelines    bool
}

// Start loads module pipelines for configured modules.
func (sr *SetupCfgRunner) Start() {
	sr.moduleRegistry.log.Debug("Loading ingest pipelines for modules from modules.d")
	pipelineLoader, err := sr.pipelineLoaderFactory()
	if err != nil {
		sr.moduleRegistry.log.Errorf("Error loading pipeline: %+v", err)
		return
	}

	err = sr.moduleRegistry.LoadPipelines(pipelineLoader, sr.overwritePipelines)
	if err != nil {
		sr.moduleRegistry.log.Errorf("Error loading pipeline: %s", err)
	}
}

// Stop of SetupCfgRunner.
func (sr *SetupCfgRunner) Stop() {}

// String returns information on the Runner
func (sr *SetupCfgRunner) String() string {
	return sr.moduleRegistry.InfoString()
}
