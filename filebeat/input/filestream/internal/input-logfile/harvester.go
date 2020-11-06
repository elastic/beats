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

package input_logfile

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"
)

// Harvester is the reader which collects the lines from
// the configured source.
type Harvester interface {
	// Name returns the type of the Harvester
	Name() string
	// Test checks if the Harvester can be started with the given configuration.
	Test(Source, input.TestContext) error
	// Run is the event loop which reads from the source
	// and forwards it to the publisher.
	Run(input.Context, Source, Cursor, Publisher) error
}

type readerGroup struct {
	mu    sync.Mutex
	table map[string]context.CancelFunc
}

func newReaderGroup() *readerGroup {
	return &readerGroup{
		table: make(map[string]context.CancelFunc),
	}
}

func (r *readerGroup) add(id string, cancel context.CancelFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.table[id]; ok {
		return fmt.Errorf("harvester is already running for file")
	}
	r.table[id] = cancel
	return nil
}

func (r *readerGroup) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cancel, ok := r.table[id]
	if !ok {
		return
	}

	cancel()
	delete(r.table, id)
}

// HarvesterGroup is responsible for running the
// Harvesters started by the Prospector.
type HarvesterGroup interface {
	Start(input.Context, Source) error
	Stop(Source)
}

type defaultHarvesterGroup struct {
	readers      *readerGroup
	locker       resourceLocker
	pipeline     beat.PipelineConnector
	harvester    Harvester
	cleanTimeout time.Duration
	store        *store
	tg           unison.TaskGroup
}

// Start starts the Harvester for a Source. It does not block.
func (hg *defaultHarvesterGroup) Start(ctx input.Context, s Source) error {
	log := ctx.Logger.With("source", s.Name())
	log.Debug("Starting harvester for file")

	go func() {
		defer func() {
			if v := recover(); v != nil {
				err := fmt.Errorf("harvester panic with: %+v\n%s", v, debug.Stack())
				ctx.Logger.Errorf("Harvester crashed with: %+v", err)
			}
		}()

		defer log.Debug("Stopped harvester for file")

		harvesterCtx, cancelHarvester := context.WithCancel(ctxtool.FromCanceller(ctx.Cancelation))
		ctx.Cancelation = harvesterCtx
		defer cancelHarvester()

		err := hg.readers.add(s.Name(), cancelHarvester)
		if err != nil {
			log.Errorf("error while adding new reader to the bookkeeper %v", err)
			return
		}
		defer hg.readers.remove(s.Name())

		resource, err := hg.locker.Lock(ctx, s.Name())
		if err != nil {
			log.Errorf("Error while locking resource: %v", err)
			return
		}
		defer releaseResource(resource)

		client, err := hg.pipeline.ConnectWith(beat.ClientConfig{
			CloseRef:   ctx.Cancelation,
			ACKHandler: newInputACKHandler(ctx.Logger),
		})
		if err != nil {
			log.Errorf("error while connecting to output with pipeline: %v", err)
			return
		}
		defer client.Close()

		hg.locker.UpdateTTL(resource, hg.cleanTimeout)
		cursor := makeCursor(hg.store, resource)
		publisher := &cursorPublisher{canceler: ctx.Cancelation, client: client, cursor: &cursor}

		err = hg.harvester.Run(ctx, s, cursor, publisher)
		if err != nil {
			log.Errorf("Harvester stopped: %v", err)
		}
	}()

	return nil
}

// Stop stops the running Harvester for a given Source.
func (hg *defaultHarvesterGroup) Stop(s Source) {
	hg.readers.remove(s.Name())
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}
