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

package beater

import (
	"io"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"

	"github.com/elastic/beats/winlogbeat/checkpoint"
	"github.com/elastic/beats/winlogbeat/eventlog"
)

type eventLogger struct {
	source     eventlog.EventLog
	eventMeta  common.EventMetadata
	processors beat.ProcessorList
}

type eventLoggerConfig struct {
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Processors           processors.PluginConfig `config:"processors"`
}

func newEventLogger(
	source eventlog.EventLog,
	options *common.Config,
) (*eventLogger, error) {
	config := eventLoggerConfig{}
	if err := options.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	return &eventLogger{
		source:     source,
		eventMeta:  config.EventMetadata,
		processors: processors,
	}, nil
}

func (e *eventLogger) connect(pipeline beat.Pipeline) (beat.Client, error) {
	api := e.source.Name()
	return pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		Processing: beat.ProcessingConfig{
			EventMetadata: e.eventMeta,
			Meta:          nil, // TODO: configure modules/ES ingest pipeline?
			Processor:     e.processors,
		},
		ACKCount: func(n int) {
			addPublished(api, n)
			logp.Info("EventLog[%s] successfully published %d events", api, n)
		},
	})
}

func (e *eventLogger) run(
	done <-chan struct{},
	pipeline beat.Pipeline,
	state checkpoint.EventLogState,
	acker *eventACKer,
) {
	api := e.source

	// Initialize per event log metrics.
	initMetrics(api.Name())

	client, err := e.connect(pipeline)
	if err != nil {
		logp.Warn("EventLog[%s] Pipeline error. Failed to connect to publisher pipeline",
			api.Name())
		return
	}

	// close client on function return or when `done` is triggered (unblock client)
	defer client.Close()
	go func() {
		<-done
		client.Close()
	}()

	err = api.Open(state)
	if err != nil {
		logp.Warn("EventLog[%s] Open() error. No events will be read from "+
			"this source. %v", api.Name(), err)
		return
	}
	defer func() {
		logp.Info("EventLog[%s] Stop processing.", api.Name())

		if err := api.Close(); err != nil {
			logp.Warn("EventLog[%s] Close() error. %v", api.Name(), err)
			return
		}
	}()

	debugf("EventLog[%s] opened successfully", api.Name())

	for stop := false; !stop; {
		select {
		case <-done:
			return
		default:
		}

		// Read from the event.
		records, err := api.Read()
		switch err {
		case nil:
		case io.EOF:
			// Graceful stop.
			stop = true
		default:
			logp.Warn("EventLog[%s] Read() error: %v", api.Name(), err)
			return
		}

		debugf("EventLog[%s] Read() returned %d records", api.Name(), len(records))
		if len(records) == 0 {
			time.Sleep(time.Second)
			continue
		}

		acker.Add(len(records))
		for _, lr := range records {
			client.Publish(lr.ToEvent())
		}
	}
}
