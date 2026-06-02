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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// commonInputConfig defines common input settings
// for the publisher pipeline.
type commonInputConfig struct {
	// event processing
	mapstr.EventMetadata `config:",inline"`      // Fields and tags to add to events.
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

// RunnerFactoryWithCommonInputSettings wraps a runner factory so all runners
// it creates share the same processing capabilities and configuration-file
// settings:
//   - *fields*: common fields to be added to the pipeline
//   - *fields_under_root*: select at which level to store the fields
//   - *tags*: add additional tags to the events
//   - *processors*: list of local processors to be added to the processing pipeline
//   - *keep_null*: keep or remove 'null' from events to be published
//   - *_module_name* (hidden setting): Add fields describing the module name
//   - *_fileset_name* (hidden setting):
//   - *pipeline*: Configure the ES Ingest Node pipeline name to be used for events from this input
//   - *index*: Configure the index name for events to be collected from this input
//   - *type*: implicit event type
//   - *service.type*: implicit event type
//
// The user-configured `processors:` list and the index processor are
// instantiated once per input and shared across all pipeline clients.
func RunnerFactoryWithCommonInputSettings(info beat.Info, f InputRunnerFactory) cfgfile.RunnerFactory {
	return &commonSettingsFactory{info: info, inner: f}
}

type commonSettingsFactory struct {
	info  beat.Info
	inner InputRunnerFactory
}

func (f *commonSettingsFactory) CheckConfig(cfg *conf.C) error {
	return f.inner.CheckConfig(cfg)
}

func (f *commonSettingsFactory) Create(pipeline beat.PipelineConnector, cfg *conf.C) (cfgfile.Runner, error) {
	editor, sharedProcs, err := newCommonConfigEditor(f.info, cfg)
	if err != nil {
		return nil, err
	}

	r, err := f.inner.Create(pipetool.WithClientConfigEdit(pipeline, editor), cfg)
	if err != nil {
		_ = sharedProcs.Close()
		return nil, err
	}

	r.AddCloser(sharedProcs)

	return r, nil
}

// InputRunner is a cfgfile.Runner that also owns the per-input shared
// processors built by newCommonConfigEditor. It closes them on Stop, after its
// pipeline clients have drained so in-flight events still see them.
type InputRunner interface {
	cfgfile.Runner

	// AddCloser hands the runner the shared processors to close on Stop.
	AddCloser(processors.Closer)
}

// InputRunnerFactory is the cfgfile.RunnerFactory variant whose runners are
// InputRunners, so RunnerFactoryWithCommonInputSettings can attach the shared
// processors without a runtime type assertion.
type InputRunnerFactory interface {
	Create(pipeline beat.PipelineConnector, cfg *conf.C) (InputRunner, error)
	CheckConfig(cfg *conf.C) error
}

// OnceSetter is implemented by runners that support `filebeat --once` (single scan then exit).
type OnceSetter interface {
	SetOnce(once bool)
}

// sharedProcessor is a run-only view of an input-owned processor. The
// per-input processors are built once and shared across every pipeline client
// the input opens (one per filestream harvester). Embedding the beat.Processor
// interface promotes only Run/String, hiding Closer and PathSetter, so a
// pipeline client closing its own processor list (when a harvester stops)
// cannot tear down or re-initialise state still used by sibling
// harvesters. The shared instances are path-initialised and closed exactly
// once by the owning input (see newCommonConfigEditor and InputRunner).
//
// This mirrors how pipeline-global processors are wrapped as a function
// processor in libbeat/publisher/processing/default.go so that clients cannot
// close them.
type sharedProcessor struct {
	beat.Processor
}

// newCommonConfigEditor builds the per-client editor closure plus the shared
// per-input processors that the editor's clients reference. The shared
// processors are returned separately so the caller closes them at input
// shutdown rather than at first-client shutdown.
func newCommonConfigEditor(
	beatInfo beat.Info,
	cfg *conf.C,
) (pipetool.ConfigEditor, *processors.Processors, error) {
	config := commonInputConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	// Build the user-configured processors once per input
	userProcessors, err := processors.New(config.Processors, beatInfo.Logger)
	if err != nil {
		return nil, nil, err
	}

	return newConfigEditor(beatInfo, config, userProcessors)
}

// newConfigEditor wires the resolved input config and the input-owned user
// processors into the per-client editor closure and the shared per-input
// processor list. It is split out from newCommonConfigEditor so the
// registry-driven processors.New build stays separate from the wiring, which
// lets tests inject processors directly instead of registering a global plugin.
func newConfigEditor(
	beatInfo beat.Info,
	config commonInputConfig,
	userProcs *processors.Processors,
) (pipetool.ConfigEditor, *processors.Processors, error) {
	serviceType := config.ServiceType
	if serviceType == "" {
		serviceType = config.Module
	}

	var indexProc beat.Processor
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err := fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			_ = userProcs.Close()
			return nil, nil, fmt.Errorf("failed to build the index processor: %w", err)
		}
		indexProc = add_formatted_index.New(timestampFormat)
	}

	shared := processors.NewList(beatInfo.Logger)
	if indexProc != nil {
		shared.AddProcessor(indexProc)
	}
	shared.AddProcessors(*userProcs)

	// Path-aware processors (cache, script, conditionals with path-aware
	// children, ...) must have their paths set before Run, otherwise
	// SafeProcessor.Run returns ErrPathsNotSet and drops every event. The
	// per-client publisher group can no longer do this for us because the
	// sharedProcessor wrapper hides PathSetter, so we initialise the shared
	// instances once here. The beat paths are already configured at startup,
	// before any input is created.
	if err := shared.SetPaths(beatInfo.Paths); err != nil {
		_ = shared.Close()
		return nil, nil, fmt.Errorf("failed to set paths for input processors: %w", err)
	}

	editor := func(clientCfg beat.ClientConfig) (beat.ClientConfig, error) {
		meta := clientCfg.Processing.Meta.Clone()
		fields := clientCfg.Processing.Fields.Clone()

		setOptional(meta, "pipeline", config.Pipeline)
		setOptional(fields, "fileset.name", config.Fileset)
		setOptional(fields, "service.type", serviceType)
		if !clientCfg.Processing.DisableType {
			setOptional(fields, "input.type", config.Type)
		}
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
		//
		// Shared processors are wrapped per client (sharedProcessor) and kept
		// as flat siblings to preserve the client group's continue-on-error
		// semantics (see TestProcessorsForConfigIsFlat).
		procs := processors.NewList(beatInfo.Logger)
		if indexProc != nil {
			procs.AddProcessor(sharedProcessor{indexProc})
		}
		if lst := clientCfg.Processing.Processor; lst != nil {
			procs.AddProcessor(lst)
		}
		for _, p := range userProcs.List {
			procs.AddProcessor(sharedProcessor{p})
		}

		clientCfg.Processing.EventMetadata = config.EventMetadata
		clientCfg.Processing.Meta = meta
		clientCfg.Processing.Fields = fields
		clientCfg.Processing.Processor = procs
		clientCfg.Processing.KeepNull = config.KeepNull
		clientCfg.Processing.DisableHost = config.PublisherPipeline.DisableHost

		return clientCfg, nil
	}

	return editor, shared, nil
}

func setOptional(to mapstr.M, key string, value string) {
	if value != "" {
		_, _ = to.Put(key, value)
	}
}
