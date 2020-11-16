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
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
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

// newContext createas a new context, cancel function and adds it to the readers list.
// the returned context does not have to be cancelled if the function returns an error.
func (r *readerGroup) newContext(id string, cancelation v2.Canceler) (context.Context, context.CancelFunc, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.table[id]; ok {
		return nil, nil, fmt.Errorf("harvester is already running for file")
	}

	ctx, cancel := context.WithCancel(ctxtool.FromCanceller(cancelation))

	r.table[id] = cancel
	return ctx, cancel, nil
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
	// Start starts a Harvester and adds it to the readers list.
	Start(input.Context, Source)
	// Stop cancels the reader of a given Source.
	Stop(Source)
	// StopGroup cancels all running Harvesters.
	StopGroup() error
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
func (hg *defaultHarvesterGroup) Start(ctx input.Context, s Source) {
	log := ctx.Logger.With("source", s.Name())
	log.Debug("Starting harvester for file")

	hg.tg.Go(func(canceler unison.Canceler) error {
		defer func() {
			if v := recover(); v != nil {
				err := fmt.Errorf("harvester panic with: %+v\n%s", v, debug.Stack())
				ctx.Logger.Errorf("Harvester crashed with: %+v", err)
			}
		}()
		defer log.Debug("Stopped harvester for file")

		harvesterCtx, cancelHarvester, err := hg.readers.newContext(s.Name(), canceler)
		if err != nil {
			return fmt.Errorf("error while adding new reader to the bookkeeper %v", err)
		}
		ctx.Cancelation = harvesterCtx
		defer cancelHarvester()
		defer hg.readers.remove(s.Name())

		resource, err := hg.locker.Lock(ctx, s.Name())
		if err != nil {
			return fmt.Errorf("error while locking resource: %v", err)
		}
		defer releaseResource(resource)

		client, err := hg.pipeline.ConnectWith(beat.ClientConfig{
			CloseRef:   ctx.Cancelation,
			ACKHandler: newInputACKHandler(ctx.Logger),
		})
		if err != nil {
			return fmt.Errorf("error while connecting to output with pipeline: %v", err)
		}
		defer client.Close()

		hg.locker.UpdateTTL(resource, hg.cleanTimeout)
		cursor := makeCursor(hg.store, resource)
		publisher := &cursorPublisher{canceler: ctx.Cancelation, client: client, cursor: &cursor}

		err = hg.harvester.Run(ctx, s, cursor, publisher)
		if err != nil {
			return fmt.Errorf("error while running harvester: %v", err)
		}
		return nil
	})
}

// Stop stops the running Harvester for a given Source.
func (hg *defaultHarvesterGroup) Stop(s Source) {
	hg.tg.Go(func(_ unison.Canceler) error {
		hg.readers.remove(s.Name())
		return nil
	})
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}

func (hg *defaultHarvesterGroup) StopGroup() error {
	return hg.tg.Stop()
}
