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
	"sync"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/processors/util"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"
)

// RunnerFactory that can be used to create cfg.Runner cast versions of Monitor
// suitable for config reloading.
type RunnerFactory struct {
	info                  beat.Info
	addTask               scheduler.AddTask
	stateLoader           monitorstate.StateLoader
	byId                  map[string]*Monitor
	mtx                   *sync.Mutex
	pluginsReg            *plugin.PluginsReg
	logger                *logp.Logger
	pipelineClientFactory PipelineClientFactory
	beatLocation          *config.LocationWithID
}

type PipelineClientFactory func(pipeline beat.Pipeline) (beat.Client, error)

type publishSettings struct {
	// Fields and tags to add to monitor.
	EventMetadata mapstr.EventMetadata    `config:",inline"`
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

type FactoryParams struct {
	BeatInfo              beat.Info
	AddTask               scheduler.AddTask
	StateLoader           monitorstate.StateLoader
	PluginsReg            *plugin.PluginsReg
	PipelineClientFactory PipelineClientFactory
	BeatRunFrom           *config.LocationWithID
}

// NewFactory takes a scheduler and creates a RunnerFactory that can create cfgfile.Runner(Monitor) objects.
func NewFactory(fp FactoryParams) *RunnerFactory {
	return &RunnerFactory{
		info:                  fp.BeatInfo,
		addTask:               fp.AddTask,
		byId:                  map[string]*Monitor{},
		mtx:                   &sync.Mutex{},
		pluginsReg:            fp.PluginsReg,
		logger:                logp.L(),
		pipelineClientFactory: fp.PipelineClientFactory,
		beatLocation:          fp.BeatRunFrom,
		stateLoader:           fp.StateLoader,
	}
}

type NoopRunner struct{}

func (NoopRunner) String() string {
	return "<noop runner>"
}

func (NoopRunner) Start() {
}

func (NoopRunner) Stop() {
}

// Create makes a new Runner for a new monitor with the given Config.
func (f *RunnerFactory) Create(p beat.Pipeline, c *conf.C) (cfgfile.Runner, error) {
	c, err := stdfields.UnnestStream(c)
	if err != nil {
		return nil, err
	}

	if !c.Enabled() {
		return NoopRunner{}, nil
	}

	configEditor, err := newCommonPublishConfigs(f.info, f.beatLocation, c)
	if err != nil {
		return nil, err
	}

	p = pipetool.WithClientConfigEdit(p, configEditor)

	f.mtx.Lock()
	defer f.mtx.Unlock()

	// This is a callback executed on stop of a monitor, it ensures we delete the entry in
	// byId.
	// It's a little tricky, because it handles the problem of this function being
	// occasionally invoked twice in one stack.
	// f.mtx would be locked given that golang does not support reentrant locks.
	// The important thing is clearing the map, not ensuring it stops exactly on time
	// so we can defer its removal from the map with a goroutine, thus breaking out of the current stack
	// and ensuring the cleanup happen soon enough.
	safeStop := func(m *Monitor) {
		go func() {
			// We can safely relock now, since we're in a new goroutine.
			f.mtx.Lock()
			defer f.mtx.Unlock()

			// If this element hasn't already been removed or replaced with a new
			// instance delete it from the map. Check monitor identity via pointer equality.
			if curM, ok := f.byId[m.stdFields.ID]; ok && curM == m {
				delete(f.byId, m.stdFields.ID)
			}
		}()
	}
	pc, err := f.pipelineClientFactory(p)
	if err != nil {
		return nil, fmt.Errorf("could not create pipeline client via factory: %w", err)
	}

	// The state loader needs the beat location to accurately load the last state
	sf, err := stdfields.ConfigToStdMonitorFields(c)
	if err != nil {
		return nil, fmt.Errorf("could not load stdfields in factory: %w", err)
	}
	loc := getLocation(f.beatLocation, sf)
	if loc != nil {
		geoMap, _ := util.GeoConfigToMap(loc.Geo)
		err = c.Merge(map[string]interface{}{
			"run_from": map[string]interface{}{
				"id":  loc.ID,
				"geo": geoMap,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("could not merge location into monitor map: %w", err)
		}
	}

	monitor, err := newMonitor(c, f.pluginsReg, pc, f.addTask, f.stateLoader, safeStop)
	if err != nil {
		return nil, fmt.Errorf("factory could not create monitor: %w", err)
	}

	if mon, ok := f.byId[monitor.stdFields.ID]; ok {
		f.logger.Warnf("monitor ID %s is configured for multiple monitors! IDs should be unique values, last seen config will win", monitor.stdFields.ID)
		// Stop the old monitor, since we'll swap our new one in place
		mon.Stop()
	}

	f.byId[monitor.stdFields.ID] = monitor

	return monitor, nil
}

// CheckConfig checks to see if the given monitor config is valid.
func (f *RunnerFactory) CheckConfig(config *conf.C) error {
	if !config.Enabled() {
		return nil
	}
	return checkMonitorConfig(config, plugin.GlobalPluginsReg)
}

// getLocation returns the location either from the stdfields or the beat preferring stdfields. Returns nil if declared in neither spot.
func getLocation(beatLocation *config.LocationWithID, sf stdfields.StdMonitorFields) (loc *config.LocationWithID) {
	// Use the monitor-specific location if possible, otherwise use the beat's location
	// Generally speaking direct HB users would use the beat location, and the synthetics service may as well (TBD)
	// while Fleet configured monitors will always use a per location monitor
	if sf.RunFrom != nil {
		loc = sf.RunFrom
	} else {
		loc = beatLocation
	}
	return loc
}

func newCommonPublishConfigs(info beat.Info, beatLocation *config.LocationWithID, cfg *conf.C) (pipetool.ConfigEditor, error) {
	var settings publishSettings
	if err := cfg.Unpack(&settings); err != nil {
		return nil, err
	}

	sf, err := stdfields.ConfigToStdMonitorFields(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not parse cfg for datastream %w", err)
	}

	// Early stage processors for setting data_stream, event.dataset, and index to write to
	loc := getLocation(beatLocation, sf)

	preProcs, err := preProcessors(info, loc, settings, sf.Type)
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
			_, _ = meta.Put("pipeline", settings.Pipeline)
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

var geoErrOnce = &sync.Once{}

// preProcessors sets up the required geo, event.dataset, data_stream.*, and write index processors for future event publishes.
func preProcessors(info beat.Info, location *config.LocationWithID, settings publishSettings, monitorType string) (procs *processors.Processors, err error) {
	procs = processors.NewList(nil)

	var dataset string
	if settings.DataStream != nil && settings.DataStream.Dataset != "" {
		dataset = settings.DataStream.Dataset
	} else {
		dataset = monitorType
	}

	// Always set event.dataset
	procs.AddProcessor(actions.NewAddFields(mapstr.M{"event": mapstr.M{"dataset": dataset}}, true, true))

	// If we have a location to add, use the add_observer_metadata processor
	if location != nil {
		var geoM mapstr.M

		geoM, err := util.GeoConfigToMap(location.Geo)
		if err != nil {
			geoErrOnce.Do(func() {
				logp.L().Warnf("could not add heartbeat geo info: %w", err)
			})
		}

		obsFields := mapstr.M{
			"observer": mapstr.M{
				"name": location.ID,
				"geo":  geoM,
			},
		}

		procs.AddProcessor(actions.NewAddFields(obsFields, true, true))
	}

	// always use synthetics data streams for browser monitors, there is no good reason not to
	// the default `heartbeat` data stream won't split out network and screenshot data.
	// at some point we should make all monitors use the `synthetics` datastreams and retire
	// the heartbeat one, but browser is the only beta one, and it would be a breaking change
	// to do so otherwise.
	if monitorType == "browser" && settings.DataStream == nil {
		settings.DataStream = &add_data_stream.DataStream{}
	}

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
		logp.L().Warn("Deprecated use of 'index' setting in heartbeat monitor, use 'data_stream' instead!")
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
