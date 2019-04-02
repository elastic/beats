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
	"fmt"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/journalbeat/reader"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

// Input manages readers and forwards entries from journals.
type Input struct {
	readers    []*reader.Reader
	done       chan struct{}
	config     Config
	pipeline   beat.Pipeline
	client     beat.Client
	states     map[string]checkpoint.JournalState
	id         uuid.UUID
	logger     *logp.Logger
	eventMeta  common.EventMetadata
	processors beat.ProcessorList
}

// New returns a new Inout
func New(
	c *common.Config,
	pipeline beat.Pipeline,
	done chan struct{},
	states map[string]checkpoint.JournalState,
) (*Input, error) {
	config := DefaultConfig
	if err := c.Unpack(&config); err != nil {
		return nil, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("error while generating ID for input: %v", err)
	}

	logger := logp.NewLogger("input").With("id", id)

	var readers []*reader.Reader
	if len(config.Paths) == 0 {
		cfg := reader.Config{
			Path:               reader.LocalSystemJournalID, // used to identify the state in the registry
			Backoff:            config.Backoff,
			MaxBackoff:         config.MaxBackoff,
			Seek:               config.Seek,
			CursorSeekFallback: config.CursorSeekFallback,
			Matches:            config.Matches,
		}

		state := states[reader.LocalSystemJournalID]
		r, err := reader.NewLocal(cfg, done, state, logger)
		if err != nil {
			return nil, fmt.Errorf("error creating reader for local journal: %v", err)
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
		}
		state := states[p]
		r, err := reader.New(cfg, done, state, logger)
		if err != nil {
			return nil, fmt.Errorf("error creating reader for journal: %v", err)
		}
		readers = append(readers, r)
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	logger.Debugf("New input is created for paths %v", config.Paths)

	return &Input{
		readers:    readers,
		done:       done,
		config:     config,
		pipeline:   pipeline,
		states:     states,
		id:         id,
		logger:     logger,
		eventMeta:  config.EventMetadata,
		processors: processors,
	}, nil
}

// Run connects to the output, collects entries from the readers
// and then publishes the events.
func (i *Input) Run() {
	var err error
	i.client, err = i.pipeline.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		Processing: beat.ProcessingConfig{
			EventMetadata: i.eventMeta,
			Meta:          nil,
			Processor:     i.processors,
		},
		ACKCount: func(n int) {
			i.logger.Infof("journalbeat successfully published %d events", n)
		},
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

			for {
				select {
				case <-i.done:
					return
				default:
				}

				event, err := r.Next()
				if event == nil {
					if err != nil {
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
