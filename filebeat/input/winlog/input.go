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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/ctxtool"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
	conf "github.com/elastic/elastic-agent-libs/config"
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

type winlogInput struct{}

// Plugin create a stateful input Plugin collecting logs from Windows Event Logs.
func Plugin(log *logp.Logger, store statestore.States) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Beta,
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

func configure(cfg *conf.C, _ *logp.Logger) ([]cursor.Source, cursor.Input, error) {
	// TODO: do we want to allow to read multiple eventLogs using a single config
	//       as is common for other inputs?
	eventLog, err := eventlog.New(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new event log. %w", err)
	}

	sources := []cursor.Source{eventLog}
	return sources, winlogInput{}, nil
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
	return eventlog.Run(
		&ctx,
		ctxtool.FromCanceller(ctx.Cancelation),
		ctx.MetricsRegistry,
		api,
		initCheckpoint(log, cursor),
		&publisher{cursorPub: pub},
		log,
	)
}

func initCheckpoint(log *logp.Logger, cursor cursor.Cursor) checkpoint.EventLogState {
	var cp checkpoint.EventLogState
	if cursor.IsNew() {
		return cp
	}

	if err := cursor.Unpack(&cp); err != nil {
		log.Errorf("Reset winlog position. Failed to read checkpoint from registry: %v", err)
		return checkpoint.EventLogState{}
	}

	return cp
}
