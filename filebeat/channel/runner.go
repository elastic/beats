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
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
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
//
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
// instantiated once per input and shared across all pipeline clients (each
// filestream harvester opens its own client). Without this, expensive
// constructors (e.g. add_kubernetes_metadata's informers and goroutines) get
// multiplied by the number of files an input matches. See elastic/beats#50376.
func RunnerFactoryWithCommonInputSettings(info beat.Info, f cfgfile.RunnerFactory) cfgfile.RunnerFactory {
	return &commonSettingsFactory{info: info, inner: f}
}

type commonSettingsFactory struct {
	info  beat.Info
	inner cfgfile.RunnerFactory
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

	if len(sharedProcs.List) == 0 {
		return r, nil
	}
	return &runnerWithSharedProcessors{Runner: r, procs: sharedProcs}, nil
}

// runnerWithSharedProcessors wraps a Runner so its Stop() also closes the
// per-input shared processors built once by newCommonConfigEditor. Stop is
// safe to call more than once.
//
// Embedding cfgfile.Runner (an interface) only promotes Start/Stop/String —
// methods of the inner concrete type that aren't on the cfgfile.Runner
// interface (e.g. SetStatusReporter) would be hidden from callers doing
// runtime type assertions on the wrapper. SetStatusReporter is forwarded
// explicitly so libbeat/cfgfile/list.go can still wire status reporting to
// the underlying input runner.
type runnerWithSharedProcessors struct {
	cfgfile.Runner
	procs    *processors.Processors
	stopOnce sync.Once
}

func (r *runnerWithSharedProcessors) Stop() {
	r.stopOnce.Do(func() {
		r.Runner.Stop()
		_ = r.procs.Close()
		r.procs = nil
	})
}

func (r *runnerWithSharedProcessors) SetStatusReporter(reporter status.StatusReporter) {
	if sr, ok := r.Runner.(status.WithStatusReporter); ok {
		sr.SetStatusReporter(reporter)
	}
}

// noCloseProcessor wraps a beat.Processor whose lifecycle is owned by the
// input (not by an individual pipeline client). The wrapper:
//
//   - swallows Close(): closing the per-client processor list (e.g. when a
//     filestream harvester stops) must not tear down state still in use by
//     sibling harvesters of the same input.
//   - forwards SetPaths(): some processors (cache, javascript script,
//     conditional processors with path-aware children) rely on lazy path
//     initialisation. Without forwarding, the publisher pipeline's group
//     SetPaths would type-assert against the wrapper and miss the inner
//     processor. SafeProcessor (applied to every registered processor by
//     SafeWrap) makes this call idempotent across the multiple harvesters
//     that connect.
type noCloseProcessor struct {
	inner beat.Processor
}

func (n *noCloseProcessor) Run(event *beat.Event) (*beat.Event, error) {
	return n.inner.Run(event)
}

func (n *noCloseProcessor) String() string {
	return n.inner.String()
}

func (n *noCloseProcessor) SetPaths(p *paths.Path) error {
	if ps, ok := n.inner.(processors.PathSetter); ok {
		return ps.SetPaths(p)
	}
	return nil
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

	serviceType := config.ServiceType
	if serviceType == "" {
		serviceType = config.Module
	}

	// Build user-configured processors once per input. Some processors
	// (e.g. add_kubernetes_metadata) spin up watchers, caches, and
	// goroutines on construction; doing this per ConnectWith call would
	// multiply the cost by the harvester count. See elastic/beats#50376.
	userProcs, err := processors.New(config.Processors, beatInfo.Logger)
	if err != nil {
		return nil, nil, err
	}

	// Build the index processor once too: its config is per-input static.
	var indexProc beat.Processor
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err := fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			_ = userProcs.Close()
			return nil, nil, fmt.Errorf("failed to build index processor: %w", err)
		}
		indexProc = add_formatted_index.New(timestampFormat)
	}

	// shared bundles every processor whose lifecycle is owned by the
	// input (closed at runner.Stop, not at client.Close). Order is not
	// meaningful here; it's just the set we must release.
	shared := processors.NewList(beatInfo.Logger)
	if indexProc != nil {
		shared.AddProcessor(indexProc)
	}
	shared.AddProcessors(*userProcs)

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
		// Per-input shared processors are wrapped in noCloseProcessor so
		// client.Close() does not tear down state used by sibling clients.
		procs := processors.NewList(beatInfo.Logger)
		if indexProc != nil {
			procs.AddProcessor(&noCloseProcessor{inner: indexProc})
		}
		if lst := clientCfg.Processing.Processor; lst != nil {
			procs.AddProcessor(lst)
		}
		for _, p := range userProcs.List {
			procs.AddProcessor(&noCloseProcessor{inner: p})
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
