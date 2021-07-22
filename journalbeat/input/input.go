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

package input

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/elastic/beats/v7/libbeat/processors/add_formatted_index"
	"github.com/elastic/go-concert/timed"

	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"

	"github.com/elastic/beats/v7/journalbeat/checkpoint"
	"github.com/elastic/beats/v7/journalbeat/reader"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
)

// Input manages readers and forwards entries from journals.
type Input struct {
	readers    []*reader.Reader
	done       chan struct{}
	config     Config
	client     beat.Client
	states     map[string]checkpoint.JournalState
	logger     *logp.Logger
	eventMeta  common.EventMetadata
	processors beat.ProcessorList
}

// New returns a new Inout
func New(
	c *common.Config,
	info beat.Info,
	done chan struct{},
	states map[string]checkpoint.JournalState,
) (*Input, error) {
	config := DefaultConfig
	if err := c.Unpack(&config); err != nil {
		return nil, err
	}

	logger := logp.NewLogger("input")
	if config.ID != "" {
		logger = logger.With("id", config.ID)
	}

	var readers []*reader.Reader
	if len(config.Paths) == 0 {
		cfg := reader.Config{
			Path:               reader.LocalSystemJournalID, // used to identify the state in the registry
			Backoff:            config.Backoff,
			MaxBackoff:         config.MaxBackoff,
			Seek:               config.Seek,
			CursorSeekFallback: config.CursorSeekFallback,
			Matches:            config.Matches,
			SaveRemoteHostname: config.SaveRemoteHostname,
			CheckpointID:       checkpointID(config.ID, reader.LocalSystemJournalID),
		}

		state := states[cfg.CheckpointID]
		r, err := reader.NewLocal(cfg, done, state, logger)
		if err != nil {
			return nil, fmt.Errorf("error creating reader for local journal: %+v", err)
		}
		readers = append(readers, r)
	}

	for _, p := range config.Paths {
		cfg := reader.Config{
			Path:               p,
			Backoff:            config.Backoff,
			MaxBackoff:         config.MaxBackoff,
			Seek:               config.Seek,
			CursorSeekFallback: config.CursorSeekFallback,
			Matches:            config.Matches,
			SaveRemoteHostname: config.SaveRemoteHostname,
			CheckpointID:       checkpointID(config.ID, p),
		}

		state := states[cfg.CheckpointID]
		r, err := reader.New(cfg, done, state, logger)
		if err != nil {
			return nil, fmt.Errorf("error creating reader for journal: %+v", err)
		}
		readers = append(readers, r)
	}

	inputProcessors, err := processorsForInput(info, config)
	if err != nil {
		return nil, err
	}

	logger.Debugf("New input is created for paths %v", config.Paths)

	return &Input{
		readers:    readers,
		done:       done,
		config:     config,
		states:     states,
		logger:     logger,
		eventMeta:  config.EventMetadata,
		processors: inputProcessors,
	}, nil
}

// Run connects to the output, collects entries from the readers
// and then publishes the events.
func (i *Input) Run(pipeline beat.Pipeline) {
	var err error
	i.client, err = pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		Processing: beat.ProcessingConfig{
			EventMetadata: i.eventMeta,
			Meta:          nil,
			Processor:     i.processors,
		},
		ACKHandler: acker.Counting(func(n int) {
			i.logger.Debugw("journalbeat successfully published events", "event.count", n)
		}),
	})
	if err != nil {
		i.logger.Error("Error connecting to output: %v", err)
		return
	}

	i.publishAll()
}

// publishAll reads events from all readers and publishes them.
func (i *Input) publishAll() {
	out := make(chan *beat.Event)
	defer close(out)

	var wg sync.WaitGroup
	defer wg.Wait()
	for _, r := range i.readers {
		wg.Add(1)
		r := r
		go func() {
			defer wg.Done()

			suppressed := atomic.NewBool(false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			for {
				select {
				case <-i.done:
					return
				default:
				}

				event, err := r.Next()
				if event == nil {
					if err != nil {
						if i.isErrSuppressed(ctx, err, suppressed) {
							i.logger.Debugf("Error message suppressed: EBADMSG")
							continue
						}
						i.logger.Errorf("Error while reading event: %v", err)
					}
					continue
				}

				select {
				case <-i.done:
				case out <- event:
				}
			}
		}()
	}

	for {
		select {
		case <-i.done:
			return
		case e := <-out:
			i.client.Publish(*e)
		}
	}
}

// isErrSuppressed checks if the error is due to a corrupt journal. If yes, only the first error message
// is displayed and then it is suppressed for 5 seconds.
func (i *Input) isErrSuppressed(ctx context.Context, err error, suppressed *atomic.Bool) bool {
	if strings.Contains(err.Error(), syscall.EBADMSG.Error()) {
		if suppressed.Load() {
			return true
		}

		suppressed.Store(true)
		go func(ctx context.Context, suppressed *atomic.Bool) {
			if err := timed.Wait(ctx, 5*time.Second); err == nil {
				suppressed.Store(false)
			}

		}(ctx, suppressed)
	}

	return false
}

// Stop stops all readers of the input.
func (i *Input) Stop() {
	for _, r := range i.readers {
		r.Close()
	}
	i.client.Close()
}

// Wait waits until all readers are done.
func (i *Input) Wait() {
	i.Stop()
}

func processorsForInput(beatInfo beat.Info, config Config) (*processors.Processors, error) {
	procs := processors.NewList(nil)

	// Processor ordering is important:
	// 1. Index configuration
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

	// 2. User processors
	userProcessors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}
	procs.AddProcessors(*userProcessors)

	return procs, nil
}

// checkpointID returns the identifier used to track persistent state for the
// input.
func checkpointID(id, path string) string {
	if id == "" {
		return path
	}
	return "journald::" + path + "::" + id
}
