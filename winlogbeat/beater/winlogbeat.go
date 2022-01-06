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

/*
Package beater provides the implementation of the libbeat Beater interface for
Winlogbeat. The main event loop is implemented in this package.
*/
package beater

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/winlogbeat/module"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/config"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
)

const pipelinesWarning = "Winlogbeat is unable to load the ingest pipelines" +
	" because the Elasticsearch output is not configured/enabled. If you have" +
	" already loaded the ingest pipelines, you can ignore this warning."

// Time the application was started.
var startTime = time.Now().UTC()

// Winlogbeat is used to conform to the beat interface
type Winlogbeat struct {
	beat       *beat.Beat              // Common beat information.
	config     config.WinlogbeatConfig // Configuration settings.
	eventLogs  []*eventLogger          // List of all event logs being monitored.
	done       chan struct{}           // Channel to initiate shutdown of main event loop.
	pipeline   beat.Pipeline           // Interface to publish event.
	checkpoint *checkpoint.Checkpoint  // Persists event log state to disk.
	log        *logp.Logger
}

// New returns a new Winlogbeat.
func New(b *beat.Beat, _ *common.Config) (beat.Beater, error) {
	// Read configuration.
	config := config.DefaultSettings
	if err := b.BeatConfig.Unpack(&config); err != nil {
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	log := logp.NewLogger("winlogbeat")

	// resolve registry file path
	config.RegistryFile = paths.Resolve(paths.Data, config.RegistryFile)
	log.Infof("State will be read from and persisted to %s",
		config.RegistryFile)

	eb := &Winlogbeat{
		beat:   b,
		config: config,
		done:   make(chan struct{}),
		log:    log,
	}

	if err := eb.init(b); err != nil {
		return nil, err
	}

	return eb, nil
}

func (eb *Winlogbeat) init(b *beat.Beat) error {
	config := &eb.config

	// Create the event logs. This will validate the event log specific
	// configuration.
	eb.eventLogs = make([]*eventLogger, 0, len(config.EventLogs))
	for _, config := range config.EventLogs {
		eventLog, err := eventlog.New(config)
		if err != nil {
			return fmt.Errorf("failed to create new event log: %w", err)
		}
		eb.log.Debugf("Initialized EventLog]", eventLog.Name())

		logger, err := newEventLogger(b.Info, eventLog, config, eb.log)
		if err != nil {
			return fmt.Errorf("failed to create new event log: %w", err)
		}

		eb.eventLogs = append(eb.eventLogs, logger)
	}
	b.OverwritePipelinesCallback = func(esConfig *common.Config) error {
		overwritePipelines := config.OverwritePipelines
		esClient, err := eslegclient.NewConnectedClient(esConfig, "Winlogbeat")
		if err != nil {
			return err
		}
		return module.UploadPipelines(b.Info, esClient, overwritePipelines)
	}
	return nil
}

// Setup uses the loaded config and creates necessary markers and environment
// settings to allow the beat to be used.
func (eb *Winlogbeat) setup(b *beat.Beat) error {
	config := &eb.config

	var err error
	eb.checkpoint, err = checkpoint.NewCheckpoint(config.RegistryFile, config.RegistryFlush)
	if err != nil {
		return err
	}

	eb.pipeline = b.Publisher
	return nil
}

// Run is used within the beats interface to execute the Winlogbeat workers.
func (eb *Winlogbeat) Run(b *beat.Beat) error {
	if err := eb.setup(b); err != nil {
		return err
	}

	if b.Config.Output.Name() == "elasticsearch" {
		callback := func(esClient *eslegclient.Connection) error {
			return module.UploadPipelines(b.Info, esClient, eb.config.OverwritePipelines)
		}
		_, err := elasticsearch.RegisterConnectCallback(callback)
		if err != nil {
			return err
		}
	} else {
		eb.log.Warn(pipelinesWarning)
	}

	acker := newEventACKer(eb.checkpoint)
	persistedState := eb.checkpoint.States()

	// Initialize metrics.
	initMetrics("total")

	var wg sync.WaitGroup
	for _, log := range eb.eventLogs {
		state, _ := persistedState[log.source.Name()]

		// Start a goroutine for each event log.
		wg.Add(1)
		go eb.processEventLog(&wg, log, state, acker)
	}

	wg.Wait()
	defer eb.checkpoint.Shutdown()

	if eb.config.ShutdownTimeout > 0 {
		eb.log.Infof("Shutdown will wait max %v for the remaining %v events to publish.",
			eb.config.ShutdownTimeout, acker.Active())
		ctx, cancel := context.WithTimeout(context.Background(), eb.config.ShutdownTimeout)
		defer cancel()
		acker.Wait(ctx)
	}

	return nil
}

// Stop is used to tell the winlogbeat that it should cease executing.
func (eb *Winlogbeat) Stop() {
	eb.log.Info("Stopping Winlogbeat")
	if eb.done != nil {
		close(eb.done)
	}
}

func (eb *Winlogbeat) processEventLog(
	wg *sync.WaitGroup,
	logger *eventLogger,
	state checkpoint.EventLogState,
	acker *eventACKer,
) {
	defer wg.Done()
	logger.run(eb.done, eb.pipeline, state, acker)
}
