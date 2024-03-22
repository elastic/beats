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
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/filestream/internal/task"
	inputv2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
)

var (
	ErrHarvesterAlreadyRunning = errors.New("harvester is already running for file")
	ErrHarvesterLimitReached   = errors.New("harvester limit reached")
)

// Harvester is the reader which collects the lines from
// the configured source.
type Harvester interface {
	// Name returns the type of the Harvester
	Name() string
	// Test checks if the Harvester can be started with the given configuration.
	Test(Source, inputv2.TestContext) error
	// Run is the event loop which reads from the source
	// and forwards it to the publisher.
	Run(inputv2.Context, Source, Cursor, Publisher, *Metrics) error
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

// newContext creates a new context, cancel function and associates it with the given id within
// the reader group. Using the cancel function does not remvoe the association.
// An error is returned if the id is already associated with a context. The cancel
// function is nil in that case and must not be called.
//
// The context will be automatically cancelled once the ID is removed from the group. Calling `cancel` is optional.
func (r *readerGroup) newContext(id string, cancelation inputv2.Canceler) (context.Context, context.CancelFunc, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.table[id]; ok {
		return nil, nil, ErrHarvesterAlreadyRunning
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

func (r *readerGroup) hasID(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.table[id]
	return ok
}

// HarvesterGroup is responsible for running the
// Harvesters started by the Prospector.
type HarvesterGroup interface {
	// Start starts a Harvester and adds it to the readers list.
	Start(inputv2.Context, Source)
	// Restart starts a Harvester if it might be already running.
	Restart(inputv2.Context, Source)
	// Continue starts a new Harvester with the state information of the previous.
	Continue(ctx inputv2.Context, previous, next Source)
	// Stop cancels the reader of a given Source.
	Stop(Source)
	// StopHarvesters cancels all running Harvesters.
	StopHarvesters() error
}

type defaultHarvesterGroup struct {
	readers      *readerGroup
	pipeline     beat.PipelineConnector
	harvester    Harvester
	cleanTimeout time.Duration
	store        *store
	ackCH        *updateChan
	identifier   *sourceIdentifier
	tg           *task.Group
	metrics      *Metrics
}

// Start starts the Harvester for a Source if no Harvester is running for the
// Source.
// If the harvester limit has been reached, the harvester will wait until it can
// be started. Start does not block.
func (hg *defaultHarvesterGroup) Start(ctx inputv2.Context, src Source) {
	sourceName := hg.identifier.ID(src)

	ctx.Logger = ctx.Logger.With("source_file", sourceName)
	ctx.Logger.Debug("Starting harvester for file")

	if err := hg.tg.Go(startHarvester(ctx, hg, src, false, hg.metrics)); err != nil {
		ctx.Logger.Warnf(
			"tried to start harvester with task group already closed",
			ctx.ID)
	}
}

// Restart starts the Harvester for a Source if a Harvester is already running
// it waits for it to shut down for a specified timeout. It does not block.
// If the harvester limit has been reached, the harvester will wait until it can
// be started. Restart does not block.
func (hg *defaultHarvesterGroup) Restart(ctx inputv2.Context, src Source) {
	sourceName := hg.identifier.ID(src)

	ctx.Logger = ctx.Logger.With("source_file", sourceName)
	ctx.Logger.Debug("Restarting harvester for file")

	if err := hg.tg.Go(startHarvester(ctx, hg, src, true, hg.metrics)); err != nil {
		ctx.Logger.Warnf(
			"input %s tried to restart harvester with task group already closed",
			ctx.ID)
	}
}

// startHarvester start starts the harvester. if restart is true, it'll first remove the
// associated reader.
// startHarvester does NOT check if the harvester limit has been reached. Its caller
// is responsible for doing so.
func startHarvester(
	ctx inputv2.Context,
	hg *defaultHarvesterGroup,
	src Source,
	restart bool,
	metrics *Metrics,
) func(context.Context) error {
	srcID := hg.identifier.ID(src)

	return func(canceler context.Context) error {
		defer func() {
			if v := recover(); v != nil {
				err := fmt.Errorf("harvester panic with: %+v\n%s", v, debug.Stack())
				ctx.Logger.Errorf("Harvester crashed with: %+v", err)
				hg.readers.remove(srcID)
			}
		}()

		if restart {
			// stop previous harvester
			hg.readers.remove(srcID)
		}
		defer ctx.Logger.Debug("Stopped harvester for file")

		harvesterCtx, cancelHarvester, err := hg.readers.newContext(srcID, canceler)
		if err != nil {
			// The only possible returned error is ErrHarvesterAlreadyRunning, which is a normal
			// behaviour of the Filestream input, it's not really an error, it's just an situation.
			// If the harvester is already running we don't need to start a new one.
			// At the moment of writing even the returned error is ignored. So the
			// only real effect of this branch is to not start a second harvester.
			//
			// Currently the only places this error is checked is on task.Group and the
			// only thing it does is to log the error. So to avoid unnecessary errors,
			// we just return nil.
			if errors.Is(err, ErrHarvesterAlreadyRunning) {
				return nil
			}

			return fmt.Errorf("error while adding new reader to the bookkeeper %w", err)
		}

		ctx.Cancelation = harvesterCtx
		defer cancelHarvester()

		resource, err := lock(ctx, hg.store, srcID)
		if err != nil {
			hg.readers.remove(srcID)
			return fmt.Errorf("error while locking resource: %w", err)
		}
		defer releaseResource(resource)

		client, err := hg.pipeline.ConnectWith(beat.ClientConfig{
			// CloseRef:      ctx.Cancelation,
			EventListener: newInputACKHandler(hg.ackCH),
		})
		if err != nil {
			hg.readers.remove(srcID)
			return fmt.Errorf("error while connecting to output with pipeline: %w", err)
		}
		defer client.Close()

		hg.store.UpdateTTL(resource, hg.cleanTimeout)
		cursor := makeCursor(resource)
		publisher := &cursorPublisher{canceler: ctx.Cancelation, client: client, cursor: &cursor}

		err = hg.harvester.Run(ctx, src, cursor, publisher, metrics)
		if err != nil && !errors.Is(err, context.Canceled) {
			hg.readers.remove(srcID)
			return fmt.Errorf("error while running harvester: %w", err)
		}
		// If the context was not cancelled it means that the Harvester is stopping because of
		// some internal decision, not due to outside interaction.
		// If it is stopping itself, it must clean up the bookkeeper.
		if !errors.Is(ctx.Cancelation.Err(), context.Canceled) {
			hg.readers.remove(srcID)
		}

		return nil
	}
}

// Continue starts a new Harvester with the state information from a different Source.
func (hg *defaultHarvesterGroup) Continue(ctx inputv2.Context, previous, next Source) {
	ctx.Logger.Debugf("Continue harvester for file prev=%s, next=%s", previous.Name(), next.Name())
	prevID := hg.identifier.ID(previous)
	nextID := hg.identifier.ID(next)

	err := hg.tg.Go(func(canceler context.Context) error {
		previousResource, err := lock(ctx, hg.store, prevID)
		if err != nil {
			return fmt.Errorf("error while locking previous resource: %w", err)
		}

		// mark previous state out of date
		// so when reading starts again the offset is set to zero
		_ = hg.store.remove(prevID) // ignoring error as it can only be "not found"

		nextResource, err := lock(ctx, hg.store, nextID)
		if err != nil {
			return fmt.Errorf("error while locking next resource: %w", err)
		}
		hg.store.UpdateTTL(nextResource, hg.cleanTimeout)

		previousResource.copyInto(nextResource)
		releaseResource(previousResource)
		releaseResource(nextResource)

		hg.Start(ctx, next)
		return nil
	})
	if err != nil {
		ctx.Logger.Warnf(
			"input %s tried to Continue harvester with task group already closed",
			ctx.ID)
	}
}

// Stop stops the running Harvester for a given Source.
func (hg *defaultHarvesterGroup) Stop(s Source) {
	_ = hg.tg.Go(func(_ context.Context) error {
		hg.readers.remove(hg.identifier.ID(s))
		return nil
	})
}

// StopHarvesters stops all running Harvesters.
func (hg *defaultHarvesterGroup) StopHarvesters() error {
	return hg.tg.Stop()
}

// Lock locks a key for exclusive access and returns a resource that can be used to modify
// the cursor state and unlock the key.
func lock(ctx inputv2.Context, store *store, key string) (*resource, error) {
	resource := store.Get(key)
	err := lockResource(ctx.Logger, resource, ctx.Cancelation)
	if err != nil {
		resource.Release()
		return nil, err
	}

	resource.stateMutex.Lock()
	resource.lockedVersion = resource.version
	resource.stateMutex.Unlock()

	return resource, nil
}

func lockResource(log *logp.Logger, resource *resource, canceler inputv2.Canceler) error {
	if !resource.lock.TryLock() {
		log.Infof("Resource '%v' currently in use, waiting...", resource.key)
		err := resource.lock.LockContext(canceler)
		log.Infof("Resource '%v' finally released. Lock acquired", resource.key)
		if err != nil {
			log.Infof("Input for resource '%v' has been stopped while waiting", resource.key)
			return err
		}
	}
	return nil
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}
