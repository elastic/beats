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

package channel

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type onCreateFactory struct {
	factory cfgfile.RunnerFactory
	create  onCreateWrapper
}

type onCreateWrapper func(cfgfile.RunnerFactory, beat.PipelineConnector, *common.Config) (cfgfile.Runner, error)

// commonInputConfig defines common input settings
// for the publisher pipeline.
type commonInputConfig struct {
	// event processing
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Processors           processors.PluginConfig `config:"processors"`
	KeepNull             bool                    `config:"keep_null"`

	PublisherPipeline struct {
		DisableHost bool `config:"disable_host"` // Disable addition of host.name.
	} `config:"publisher_pipeline"`

	// implicit event fields
	Type        string `config:"type"`         // input.type
	ServiceType string `config:"service.type"` // service.type

	// hidden filebeat modules settings
	Module  string `config:"_module_name"`  // hidden setting
	Fileset string `config:"_fileset_name"` // hidden setting

	// Output meta data settings
	Pipeline string                   `config:"pipeline"` // ES Ingest pipeline name
	Index    fmtstr.EventFormatString `config:"index"`    // ES output index pattern
}

func (f *onCreateFactory) CheckConfig(cfg *common.Config) error {
	return f.factory.CheckConfig(cfg)
}

func (f *onCreateFactory) Create(pipeline beat.PipelineConnector, cfg *common.Config) (cfgfile.Runner, error) {
	return f.create(f.factory, pipeline, cfg)
}

// RunnerFactoryWithCommonInputSettings wraps a runner factory, such that all runners
// created by this factory have the same processing capabilities and related
// configuration file settings.
//
// Common settings ensured by this factory wrapper:
//  - *fields*: common fields to be added to the pipeline
//  - *fields_under_root*: select at which level to store the fields
//  - *tags*: add additional tags to the events
//  - *processors*: list of local processors to be added to the processing pipeline
//  - *keep_null*: keep or remove 'null' from events to be published
//  - *_module_name* (hidden setting): Add fields describing the module name
//  - *_ fileset_name* (hiddrn setting):
//  - *pipeline*: Configure the ES Ingest Node pipeline name to be used for events from this input
//  - *index*: Configure the index name for events to be collected from this input
//  - *type*: implicit event type
//  - *service.type*: implicit event type
func RunnerFactoryWithCommonInputSettings(info beat.Info, f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
	return wrapRunnerCreate(f,
		func(
			f cfgfile.RunnerFactory,
			pipeline beat.PipelineConnector,
			cfg *common.Config,
		) (runner cfgfile.Runner, err error) {
			pipeline, err = withClientConfig(info, pipeline, cfg)
			if err != nil {
				return nil, err
			}

			return f.Create(pipeline, cfg)
		})
}

func wrapRunnerCreate(f cfgfile.RunnerFactory, edit onCreateWrapper) cfgfile.RunnerFactory {
	return &onCreateFactory{factory: f, create: edit}
}

// withClientConfig reads common Beat input instance configurations from the
// configuration object and ensure that the settings are applied to each client.
func withClientConfig(
	beatInfo beat.Info,
	pipeline beat.PipelineConnector,
	cfg *common.Config,
) (beat.PipelineConnector, error) {
	editor, err := newCommonConfigEditor(beatInfo, cfg)
	if err != nil {
		return nil, err
	}
	return pipetool.WithClientConfigEdit(pipeline, editor), nil
}

func newCommonConfigEditor(
	beatInfo beat.Info,
	cfg *common.Config,
) (pipetool.ConfigEditor, error) {
	config := commonInputConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	var indexProcessor processors.Processor
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err := fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			return nil, err
		}
		indexProcessor = add_formatted_index.New(timestampFormat)
	}

	userProcessors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	serviceType := config.ServiceType
	if serviceType == "" {
		serviceType = config.Module
	}

	return func(clientCfg beat.ClientConfig) (beat.ClientConfig, error) {
		meta := clientCfg.Processing.Meta.Clone()
		fields := clientCfg.Processing.Fields.Clone()

		setOptional(meta, "pipeline", config.Pipeline)
		setOptional(fields, "fileset.name", config.Fileset)
		setOptional(fields, "service.type", serviceType)
		setOptional(fields, "input.type", config.Type)
		if config.Module != "" {
			event := mapstr.M{"module": config.Module}
			if config.Fileset != "" {
				event["dataset"] = config.Module + "." + config.Fileset
			}
			fields["event"] = event
		}

		// assemble the processors. Ordering is important.
		// 1. add support for index configuration via processor
		// 2. add processors added by the input that wants to connect
		// 3. add locally configured processors from the 'processors' settings
		procs := processors.NewList(nil)
		if indexProcessor != nil {
			procs.AddProcessor(indexProcessor)
		}
		if lst := clientCfg.Processing.Processor; lst != nil {
			procs.AddProcessor(lst)
		}
		if userProcessors != nil {
			procs.AddProcessors(*userProcessors)
		}

		clientCfg.Processing.EventMetadata = config.EventMetadata
		clientCfg.Processing.Meta = meta
		clientCfg.Processing.Fields = fields
		clientCfg.Processing.Processor = procs
		clientCfg.Processing.KeepNull = config.KeepNull
		clientCfg.Processing.DisableHost = config.PublisherPipeline.DisableHost

		return clientCfg, nil
	}, nil
}

func setOptional(to mapstr.M, key string, value string) {
	if value != "" {
		to.Put(key, value)
	}
}
