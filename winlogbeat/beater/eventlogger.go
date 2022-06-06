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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/beats/v7/libbeat/publisher/pipetool"

	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/beats/v7/winlogbeat/eventlog"
)

type eventLogger struct {
	source     eventlog.EventLog
	eventMeta  common.EventMetadata
	processors beat.ProcessorList
	keepNull   bool
	log        *logp.Logger
}

type eventLoggerConfig struct {
	common.EventMetadata `config:",inline"` // Fields and tags to add to events.

	Processors processors.PluginConfig  `config:"processors"`
	Index      fmtstr.EventFormatString `config:"index"`

	// KeepNull determines whether published events will keep null values or omit them.
	KeepNull bool `config:"keep_null"`
}

func newEventLogger(
	beatInfo beat.Info,
	source eventlog.EventLog,
	options *common.Config,
	log *logp.Logger,
) (*eventLogger, error) {
	config := eventLoggerConfig{}
	if err := options.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processorsForConfig(beatInfo, config)
	if err != nil {
		return nil, err
	}

	return &eventLogger{
		source:     source,
		eventMeta:  config.EventMetadata,
		processors: processors,
		log:        log.With("id", source.Name()),
	}, nil
}

func (e *eventLogger) connect(pipeline beat.Pipeline) (beat.Client, error) {
	return pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		Processing: beat.ProcessingConfig{
			EventMetadata: e.eventMeta,
			Meta:          nil, // TODO: configure modules/ES ingest pipeline?
			Processor:     e.processors,
			KeepNull:      e.keepNull,
		},
		ACKHandler: acker.Counting(func(n int) {
			addPublished(e.source.Name(), n)
			e.log.Debugw("Successfully published events.", "event.count", n)
		}),
	})
}

func (e *eventLogger) run(
	done <-chan struct{},
	pipeline beat.Pipeline,
	state checkpoint.EventLogState,
	eventACKer *eventACKer,
) {
	api := e.source

	// Initialize per event log metrics.
	initMetrics(api.Name())

	pipeline = pipetool.WithACKer(pipeline, acker.EventPrivateReporter(func(_ int, private []interface{}) {
		eventACKer.ACKEvents(private)
	}))

	client, err := e.connect(pipeline)
	if err != nil {
		e.log.Warnw("Pipeline error. Failed to connect to publisher pipeline", "error", err)
		return
	}

	// close client on function return or when `done` is triggered (unblock client)
	defer client.Close()
	go func() {
		<-done
		client.Close()
	}()

	defer func() {
		e.log.Info("Stop processing.")

		if err := api.Close(); err != nil {
			e.log.Warnw("Close() error.", "error", err)
			return
		}
	}()

runLoop:
	for stop := false; !stop; {
		err = api.Open(state)
		if eventlog.IsRecoverable(err) {
			e.log.Warnw("Open() encountered recoverable error. Trying again...", "error", err)
			time.Sleep(time.Second * 5)
			continue
		} else if err != nil {
			e.log.Warnw("Open() error. No events will be read from this source.", "error", err)
			return
		}
		e.log.Debug("Opened successfully.")

		for !stop {
			select {
			case <-done:
				return
			default:
			}

			// Read from the event.
			records, err := api.Read()
			if eventlog.IsRecoverable(err) {
				e.log.Warnw("Read() encountered recoverable error. Reopening handle...", "error", err)
				if closeErr := api.Close(); closeErr != nil {
					e.log.Warnw("Close() error.", "error", err)
				}
				continue runLoop
			}
			switch err {
			case nil:
			case io.EOF:
				// Graceful stop.
				stop = true
			default:
				e.log.Warnw("Read() error.", "error", err)
				return
			}

			e.log.Debugf("Read() returned %d records.", len(records))
			if len(records) == 0 {
				time.Sleep(time.Second)
				if stop {
					return
				}
				continue
			}

			eventACKer.Add(len(records))
			for _, lr := range records {
				client.Publish(lr.ToEvent())
			}
		}
	}
}

// processorsForConfig assembles the Processors for an eventLogger.
func processorsForConfig(
	beatInfo beat.Info, config eventLoggerConfig,
) (*processors.Processors, error) {
	procs := processors.NewList(nil)

	// Processor order is important! The index processor, if present, must be
	// added before the user processors.
	if !config.Index.IsEmpty() {
		staticFields := fmtstr.FieldsForBeat(beatInfo.Beat, beatInfo.Version)
		timestampFormat, err :=
			fmtstr.NewTimestampFormatString(&config.Index, staticFields)
		if err != nil {
			return nil, err
		}
		indexProcessor := add_formatted_index.New(timestampFormat)
		procs.AddProcessor(indexProcessor)
	}

	userProcs, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}
	procs.AddProcessors(*userProcs)

	return procs, nil
}
