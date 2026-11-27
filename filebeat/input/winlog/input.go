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

package winlog

import (
	"fmt"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/monitoring"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
)

const pluginName = "winlog"

type publisher struct {
	cursorPub cursor.Publisher
}

func (pub *publisher) Publish(records []eventlog.Record) error {
	for _, record := range records {
		event := record.ToEvent()
		if err := pub.cursorPub.Publish(event, record.Offset); err != nil {
			// Publisher indicates disconnect when returning an error.
			// stop trying to publish records and quit
			return err
		}
	}
	return nil
}

// Options customises the winlog input for hosts other than Filebeat.
type Options struct {
	// InitialState returns the position to start from for an event log that
	// has no cursor in the registry. name is the event log's Name().
	InitialState func(name string) (checkpoint.EventLogState, bool)
	// OnStopped is called once when an event log's reader returns for any
	// reason other than cancellation. name is the event log's Name().
	OnStopped func(name string)
}

type winlogInput struct {
	opts Options
}

// Plugin create a stateful input Plugin collecting logs from Windows Event Logs.
func Plugin(log *logp.Logger, store statestore.States) input.Plugin {
	return PluginWithOptions(log, store, Options{})
}

// PluginWithOptions is like Plugin but applies opts to the input.
func PluginWithOptions(log *logp.Logger, store statestore.States, opts Options) input.Plugin {
	configure := func(cfg *conf.C, logger *logp.Logger) ([]cursor.Source, cursor.Input, error) {
		// TODO: do we want to allow to read multiple eventLogs using a single config
		//       as is common for other inputs?
		eventLog, err := eventlog.New(cfg, logger)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new event log. %w", err)
		}

		sources := []cursor.Source{eventLog}
		return sources, winlogInput{opts: opts}, nil
	}

	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Windows Event Logs",
		Doc:        "The winlog input collects logs from the local windows event log service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

func (winlogInput) Name() string { return pluginName }

func (in winlogInput) Test(source cursor.Source, ctx input.TestContext) error {
	api, _ := source.(eventlog.EventLog)
	err := api.Open(checkpoint.EventLogState{}, monitoring.NewRegistry())
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", api.Channel(), err)
	}
	return api.Close()
}

func (in winlogInput) Run(
	ctx input.Context,
	source cursor.Source,
	cursor cursor.Cursor,
	pub cursor.Publisher,
) error {
	api, _ := source.(eventlog.EventLog)
	log := ctx.Logger.With("eventlog", source.Name(), "channel", api.Channel())
	return in.run(ctx, api, in.startState(log, source.Name(), cursor.IsNew(), cursor.Unpack), &publisher{cursorPub: pub})
}

// startState picks the start position: the unpacked cursor if one exists,
// else opts.InitialState, else the zero state.
func (in winlogInput) startState(log *logp.Logger, name string, isNew bool, unpack func(any) error) checkpoint.EventLogState {
	var cp checkpoint.EventLogState
	if isNew {
		if in.opts.InitialState != nil {
			if state, ok := in.opts.InitialState(name); ok {
				return state
			}
		}
		return cp
	}

	if err := unpack(&cp); err != nil {
		log.Errorf("Reset winlog position. Failed to read checkpoint from registry: %v", err)
		return checkpoint.EventLogState{}
	}

	return cp
}

// run calls eventlog.Run and then OnStopped when ctx.Cancelation is not done.
func (in winlogInput) run(ctx input.Context, api eventlog.EventLog, start checkpoint.EventLogState, pub eventlog.Publisher) error {
	log := ctx.Logger.With("eventlog", api.Name(), "channel", api.Channel())
	err := eventlog.Run(&ctx, ctx.Cancelation, ctx.MetricsRegistry, api, start, pub, log)
	if in.opts.OnStopped != nil && ctx.Cancelation.Err() == nil {
		in.opts.OnStopped(api.Name())
	}
	return err
}
