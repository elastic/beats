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
	"io"
	"time"

	input "github.com/menderesk/beats/v7/filebeat/input/v2"
	cursor "github.com/menderesk/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/feature"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/go-concert/ctxtool"
	"github.com/menderesk/go-concert/timed"

	"github.com/menderesk/beats/v7/winlogbeat/checkpoint"
	"github.com/menderesk/beats/v7/winlogbeat/eventlog"
)

type eventlogRunner struct{}

const pluginName = "winlog"

// Plugin create a stateful input Plugin collecting logs from Windows Event Logs.
func Plugin(log *logp.Logger, store cursor.StateStore) input.Plugin {
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

func configure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	// TODO: do we want to allow to read multiple eventLogs using a single config
	//       as is common for other inputs?
	eventLog, err := eventlog.New(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create new event log. %v", err)
	}

	sources := []cursor.Source{eventLog}
	return sources, eventlogRunner{}, nil
}

func (eventlogRunner) Name() string { return pluginName }

func (eventlogRunner) Test(source cursor.Source, ctx input.TestContext) error {
	api := source.(eventlog.EventLog)
	err := api.Open(checkpoint.EventLogState{})
	if err != nil {
		return fmt.Errorf("Failed to open '%v': %v", api.Name(), err)
	}
	return api.Close()
}

func (eventlogRunner) Run(
	ctx input.Context,
	source cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	log := ctx.Logger.With("eventlog", source.Name())
	api := source.(eventlog.EventLog)

	// setup closing the API if either the run function is signaled asynchronously
	// to shut down or when returning after io.EOF
	cancelCtx, cancelFn := ctxtool.WithFunc(ctx.Cancelation, func() {
		if err := api.Close(); err != nil {
			log.Errorf("Error while closing Windows Eventlog Access: %v", err)
		}
	})
	defer cancelFn()

runLoop:
	for {
		if cancelCtx.Err() != nil {
			return nil
		}

		evtCheckpoint := initCheckpoint(log, cursor)
		openErr := api.Open(evtCheckpoint)
		if eventlog.IsRecoverable(openErr) {
			log.Errorf("Encountered recoverable error when opening Windows Event Log: %v", openErr)
			timed.Wait(cancelCtx, 5*time.Second)
			continue
		} else if openErr != nil {
			return fmt.Errorf("failed to open windows event log: %v", openErr)
		}
		log.Debugf("Windows Event Log '%s' opened successfully", source.Name())

		// read loop
		for cancelCtx.Err() == nil {
			records, err := api.Read()
			if eventlog.IsRecoverable(err) {
				log.Errorf("Encountered recoverable error when reading from Windows Event Log: %v", err)
				if closeErr := api.Close(); closeErr != nil {
					log.Errorf("Error closing Windows Event Log handle: %v", closeErr)
				}
				continue runLoop
			}
			switch err {
			case nil:
				break
			case io.EOF:
				log.Debugf("End of Winlog event stream reached: %v", err)
				return nil
			default:
				// only log error if we are not shutting down
				if cancelCtx.Err() != nil {
					return nil
				}

				log.Errorf("Error occured while reading from Windows Event Log '%v': %v", source.Name(), err)
				return err
			}

			if len(records) == 0 {
				timed.Wait(cancelCtx, time.Second)
				continue
			}

			for _, record := range records {
				event := record.ToEvent()
				if err := publisher.Publish(event, record.Offset); err != nil {
					// Publisher indicates disconnect when returning an error.
					// stop trying to publish records and quit
					return err
				}
			}
		}
	}
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
