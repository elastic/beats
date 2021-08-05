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

package monitors

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
)

// RunnerFactory that can be used to create cfg.Runner cast versions of Monitor
// suitable for config reloading.
type RunnerFactory struct {
	info  beat.Info
	sched *scheduler.Scheduler
}

type publishSettings struct {
	// Fields and tags to add to monitor.
	EventMetadata common.EventMetadata    `config:",inline"`
	Processors    processors.PluginConfig `config:"processors"`

	PublisherPipeline struct {
		DisableHost bool `config:"disable_host"` // Disable addition of host.name.
	} `config:"publisher_pipeline"`

	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool `config:"keep_null"`

	// Output meta data settings
	Pipeline   string                      `config:"pipeline"` // ES Ingest pipeline name
	Index      fmtstr.EventFormatString    `config:"index"`    // ES output index pattern
	DataStream *add_data_stream.DataStream `config:"data_stream"`
	DataSet    string                      `config:"dataset"`
}

// NewFactory takes a scheduler and creates a RunnerFactory that can create cfgfile.Runner(Monitor) objects.
func NewFactory(info beat.Info, sched *scheduler.Scheduler) *RunnerFactory {
	return &RunnerFactory{info, sched}
}

// Create makes a new Runner for a new monitor with the given Config.
func (f *RunnerFactory) Create(p beat.Pipeline, c *common.Config) (cfgfile.Runner, error) {
	c, err := stdfields.UnnestStream(c)
	if err != nil {
		return nil, err
	}

	configEditor, err := newCommonPublishConfigs(f.info, c)
	if err != nil {
		return nil, err
	}

	p = pipetool.WithClientConfigEdit(p, configEditor)
	monitor, err := newMonitor(c, plugin.GlobalPluginsReg, p, f.sched)
	return monitor, err
}

// CheckConfig checks to see if the given monitor config is valid.
func (f *RunnerFactory) CheckConfig(config *common.Config) error {
	return checkMonitorConfig(config, plugin.GlobalPluginsReg)
}

func newCommonPublishConfigs(info beat.Info, cfg *common.Config) (pipetool.ConfigEditor, error) {
	var settings publishSettings
	if err := cfg.Unpack(&settings); err != nil {
		return nil, err
	}

	sf, err := stdfields.ConfigToStdMonitorFields(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse cfg for datastream %w", err)
	}

	// Early stage processors for setting data_stream, event.dataset, and index to write to
	preProcs, err := preProcessors(info, settings, sf.Type)
	if err != nil {
		return nil, err
	}

	userProcessors, err := processors.New(settings.Processors)
	if err != nil {
		return nil, err
	}

	return func(clientCfg beat.ClientConfig) (beat.ClientConfig, error) {
		fields := clientCfg.Processing.Fields.Clone()

		meta := clientCfg.Processing.Meta.Clone()
		if settings.Pipeline != "" {
			meta.Put("pipeline", settings.Pipeline)
		}

		procs := processors.NewList(nil)

		if lst := clientCfg.Processing.Processor; lst != nil {
			procs.AddProcessor(lst)
		}
		procs.AddProcessors(*preProcs)
		if userProcessors != nil {
			procs.AddProcessors(*userProcessors)
		}

		clientCfg.Processing.EventMetadata = settings.EventMetadata
		clientCfg.Processing.Fields = fields
		clientCfg.Processing.Meta = meta
		clientCfg.Processing.Processor = procs
		clientCfg.Processing.KeepNull = settings.KeepNull
		clientCfg.Processing.DisableHost = settings.PublisherPipeline.DisableHost

		return clientCfg, nil
	}, nil
}

// preProcessors sets up the required event.dataset, data_stream.*, and write index processors for future event publishes.
func preProcessors(info beat.Info, settings publishSettings, monitorType string) (procs *processors.Processors, err error) {
	procs = processors.NewList(nil)

	var dataset string
	if settings.DataStream != nil && settings.DataStream.Dataset != "" {
		dataset = settings.DataStream.Dataset
	} else {
		dataset = monitorType
	}

	// Always set event.dataset
	procs.AddProcessor(actions.NewAddFields(common.MapStr{"event": common.MapStr{"dataset": dataset}}, true, true))

	if settings.DataStream != nil {
		ds := *settings.DataStream
		if ds.Type == "" {
			ds.Type = "synthetics"
		}
		if ds.Dataset == "" {
			ds.Dataset = dataset
		}

		procs.AddProcessor(add_data_stream.New(ds))
	}

	if !settings.Index.IsEmpty() {
		proc, err := indexProcessor(&settings.Index, info)
		if err != nil {
			return nil, err
		}
		procs.AddProcessor(proc)
	}

	return procs, nil
}

func indexProcessor(index *fmtstr.EventFormatString, info beat.Info) (beat.Processor, error) {
	staticFields := fmtstr.FieldsForBeat(info.Beat, info.Version)

	timestampFormat, err :=
		fmtstr.NewTimestampFormatString(index, staticFields)
	if err != nil {
		return nil, err
	}
	return add_formatted_index.New(timestampFormat), nil
}
