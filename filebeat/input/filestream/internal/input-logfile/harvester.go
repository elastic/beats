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
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/go-concert/ctxtool"
)

var ErrHarvesterAlreadyRunning = errors.New("harvester is already running for file")

type permanentHarvesterError struct {
	err error
}

func (e permanentHarvesterError) Error() string {
	return e.err.Error()
}

func (e permanentHarvesterError) Unwrap() error {
	return e.err
}

// Harvester is the reader which collects the lines from
// the configured source.
type Harvester interface {
	// Name returns the type of the Harvester
	Name() string
	// Test checks if the Harvester can be started with the given configuration.
	Test(Source, inputv2.TestContext) error
	// Run is the event loop which reads from the source
	// and forwards it to the publisher.
	Run(inputv2.Context, Source, string, Cursor, Publisher, *Metrics) error
}

// reader is the handle for one harvester's registration in a readerGroup.
type reader struct {
	group *readerGroup
	// srcID is the current registration key, guarded by group.mu.
	srcID  string
	cancel context.CancelFunc
}

func (rd *reader) currentID() string {
	rd.group.mu.Lock()
	defer rd.group.mu.Unlock()

	return rd.srcID
}

// currentResource atomically returns the reader's registration key and the store resource for it.
func (rd *reader) currentResource(store *store) (string, *resource) {
	rd.group.mu.Lock()
	defer rd.group.mu.Unlock()

	return rd.srcID, store.Get(rd.srcID)
}

// register upgrades a reservation into a running registration and returns the harvester's context.
// It registers under the reader's current ID, which may differ from the reserved one.
func (rd *reader) register(cancelation inputv2.Canceler) (context.Context, error) {
	rd.group.mu.Lock()
	defer rd.group.mu.Unlock()

	if rd.group.table[rd.srcID] != rd {
		return nil, fmt.Errorf("reservation for %s was removed before the harvester started", rd.srcID)
	}
	if rd.cancel != nil {
		return nil, ErrHarvesterAlreadyRunning
	}

	ctx, cancel := context.WithCancel(ctxtool.FromCanceller(cancelation))
	rd.cancel = cancel
	return ctx, nil
}

// remove cancels the reader's context and deletes its registration from the group. A removed reader
// cannot register anymore.
func (rd *reader) remove() {
	rd.group.mu.Lock()
	defer rd.group.mu.Unlock()

	rd.unsafeRemove()
}

func (rd *reader) unsafeRemove() {
	if rd.cancel != nil {
		rd.cancel()
	}
	if rd.group.table[rd.srcID] == rd {
		delete(rd.group.table, rd.srcID)
	}
}

type readerGroup struct {
	mu sync.Mutex
	// table maps each source ID to its non-nil reader handle
	table map[string]*reader
}

func newReaderGroup() *readerGroup {
	return &readerGroup{
		table: make(map[string]*reader),
	}
}

func (r *readerGroup) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rd := r.table[id]; rd != nil {
		rd.unsafeRemove()
	}
}

// migrate atomically applies updateStore and re-keys any harvester registration from oldID to
// newID, keeping the same reader handle.
func (r *readerGroup) migrate(oldID, newID string, updateStore func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.table[newID]; exists {
		// Target occupied — don't clobber an existing registration.
		return fmt.Errorf("a harvester is already registered for %q", newID)
	}

	if err := updateStore(); err != nil {
		return err
	}

	rd := r.table[oldID]
	if rd == nil {
		// No harvester for oldID; only the store needed re-keying.
		return nil
	}

	delete(r.table, oldID)
	rd.srcID = newID
	r.table[newID] = rd
	return nil
}

// reserve creates a reservation for the id, to be upgraded via reader.register once the harvester
// goroutine runs. It returns nil if the id is already taken.
func (r *readerGroup) reserve(id string) *reader {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.table[id]; exists {
		return nil
	}
	rd := &reader{group: r, srcID: id}
	r.table[id] = rd
	return rd
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
	// SetObserver sets the observer to get notified when a harvester closes
	SetObserver(c chan HarvesterStatus)
	// Migrate moves a running harvester's bookkeeping registration in-place
	Migrate(oldID string, next Source, updateStore func(newID string) error) error
}

type defaultHarvesterGroup struct {
	readers      *readerGroup
	pipeline     beat.PipelineConnector
	harvester    Harvester
	cleanTimeout time.Duration
	store        *store
	ackCH        *updateChan
	identifier   *SourceIdentifier
	tg           *task.Group
	metrics      *Metrics
	notifyChan   chan HarvesterStatus
	inputID      string
	readUntilEOF ReadUntilEOFConfig
}

// HarvesterStatus is used to notify an observer that the harvester for the ID
// has closed and the amount of data ingested from the file.
type HarvesterStatus struct {
	// ID is the ID of the harvester
	ID string
	// Size is the amount of data ingested, in other words the size of the file
	// when the harvester closed.
	Size int64
}

func (hg *defaultHarvesterGroup) notifyObserver(canceler inputv2.Canceler, srcID string, size int64) {
	if hg.notifyChan == nil {
		return
	}

	select {
	case hg.notifyChan <- HarvesterStatus{srcID, size}:
	case <-canceler.Done():
	}
}

// SetObserver sets the observer to get notifications when a harvester closes
func (hg *defaultHarvesterGroup) SetObserver(c chan HarvesterStatus) {
	hg.notifyChan = c
}

// Start starts the Harvester for a Source if no Harvester is running for the
// Source.
// If the harvester limit has been reached, the harvester will wait until it can
// be started. Start does not block.
func (hg *defaultHarvesterGroup) Start(ctx inputv2.Context, src Source) {
	fn := startHarvester(ctx, hg, src, false, hg.metrics, hg.inputID)
	if fn == nil {
		return
	}

	if err := hg.tg.Go(fn); err != nil {
		ctx.Logger.Warnf(
			"tried to start harvester for %s with task group already closed",
			ctx.ID)
	}
}

// Restart starts the Harvester for a Source if a Harvester is already running
// it waits for it to shut down for a specified timeout. It does not block.
// If the harvester limit has been reached, the harvester will wait until it can
// be started. Restart does not block.
func (hg *defaultHarvesterGroup) Restart(ctx inputv2.Context, src Source) {
	ctx.Logger.Debugf("Restarting harvester for file %q", src.LogPath())

	if err := hg.tg.Go(startHarvester(ctx, hg, src, true, hg.metrics, hg.inputID)); err != nil {
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
	inputID string,
) func(context.Context) error {
	srcID := hg.identifier.ID(src)
	logPath := src.LogPath()
	// rd is this harvester's registration handle; all key-dependent steps go through it rather than
	// srcID because a migration can re-key the registration at any time.
	var rd *reader
	if !restart {
		rd = hg.readers.reserve(srcID)
		if rd == nil {
			// A harvester is already running for this source, no need to start another. This check
			// must happen here, before task.Group.Go spawns a goroutine. When harvester_limit is
			// set, the spawned goroutine blocks on a semaphore until a slot is available. Without
			// this early check, repeated file events would spawn goroutines that wait on the
			// semaphore only to discover (after acquiring it) that a harvester is already running.
			ctx.Logger.Debugf("Harvester already running for file %q", logPath)
			return nil
		}
	}

	return func(canceler context.Context) (err error) {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("harvester panic for file %q: %+v\n%s", logPath, v, debug.Stack())
				if rd != nil {
					rd.remove()
				}
			}

			// Report permanent harvester errors as a degraded state for the input.
			if err != nil && isPermanentHarvesterError(err) {
				ctx.UpdateStatus(
					status.Degraded,
					fmt.Sprintf("Harvester for Filestream input %q failed: %s", inputID, err),
				)
			}
		}()

		// We clone the logger here where we need it to avoid redundant copies that increase memory pressure.
		ctx.Logger = ctx.Logger.With("source_file", logPath)

		if restart {
			// Stop the previous harvester and take its place.
			hg.readers.remove(srcID)
			rd = hg.readers.reserve(srcID)
			if rd == nil {
				// Another Start raced in; leave the source to it.
				ctx.Logger.Debugf("Harvester already running for file %q", logPath)
				return nil
			}
		}

		harvesterCtx, err := rd.register(canceler)
		if err != nil {
			// A normal situation, not really an error: the source was stopped before this goroutine
			// got to run.
			ctx.Logger.Debugf("Harvester not started: %v", err)
			return nil
		}

		defer func() {
			if err != nil {
				ctx.Logger.Debugf("Stopped harvester for file due to an error: %s", err)
				return
			}
			ctx.Logger.Debugf("Stopped harvester for file")
		}()

		ctx.Cancelation = harvesterCtx
		defer rd.cancel()

		id, resource := rd.currentResource(hg.store)
		if err := lockResource(ctx, resource, id); err != nil {
			rd.remove()
			return fmt.Errorf("error while locking resource: %w", err)
		}
		defer releaseResource(resource)

		client, err := hg.pipeline.ConnectWith(beat.ClientConfig{
			EventListener: newInputACKHandler(hg.ackCH),
		})
		if err != nil {
			rd.remove()
			return permanentHarvesterError{
				err: fmt.Errorf("error while connecting to output with pipeline: %w", err),
			}
		}
		defer client.Close()

		hg.store.UpdateTTL(resource, hg.cleanTimeout)
		cursor := makeCursor(resource)

		// When read_until_eof is enabled the canceler must be nil. If the harvester
		// is blocked in client.Publish at the moment of input cancel, Publish
		// eventually returns once backpressure releases. With a non-nil canceler,
		// forward would then return ctx.Cancelation.Err() (context.Canceled),
		// readLineFromSource would surface it as a generic error, and
		// handleReadError would end the normal-read loop without entering the
		// readUntilEOF drain. Returning nil here lets the outer loop re-check
		// ctx.Cancelation and fall through to the readUntilEOF loop and finish
		// reading the file and publishing events until EOF or the timeout is
		// reached.
		var publisherCanceler = ctx.Cancelation
		if hg.readUntilEOF.Enabled {
			publisherCanceler = nil
		}
		publisher := &cursorPublisher{
			canceler: publisherCanceler,
			client:   client,
			cursor:   &cursor}

		defer func() {
			// The cursor struct used by Filestream, it is defined on:
			// filebeat/input/filestream/input.go
			st := struct {
				Offset int64 `json:"offset" struct:"offset"`
			}{}
			if err := cursor.Unpack(&st); err != nil {
				// Unpack should never fail, if it fails either the cursor
				// structure had a breaking change or our registry is corrupted.
				// Either way, it is better to not notify the observer.
				ctx.Logger.Errorf("cannot unpack cursor at the end of the harvester: %s", err)
				return
			}

			hg.notifyObserver(canceler, rd.currentID(), st.Offset)
			ctx.Logger.Debugf("Harvester closed with offset: %d", st.Offset)
		}()

		ctx.Logger.Debugf("Starting harvester for file. offset %v", resource.cursor)
		err = hg.harvester.Run(ctx, src, id, cursor, publisher, metrics)
		if err != nil && !errors.Is(err, context.Canceled) {
			rd.remove()
			return fmt.Errorf("error while running harvester: %w", err)
		}
		// If the context was not cancelled it means that the Harvester is stopping because of
		// some internal decision, not due to outside interaction.
		// If it is stopping itself, it must clean up the bookkeeper.
		if !errors.Is(ctx.Cancelation.Err(), context.Canceled) {
			rd.remove()
		}

		return nil
	}
}

// Continue starts a new Harvester with the state information from a different Source.
func (hg *defaultHarvesterGroup) Continue(ctx inputv2.Context, previous, next Source) {
	ctx.Logger.Debugf("Continue harvester for file, previous=%q next=%q", previous.LogPath(), next.LogPath())
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

// Stop stops the running (or pending) Harvester for a given Source. It is synchronous so that a
// Stop can never outrun a later Start.
func (hg *defaultHarvesterGroup) Stop(s Source) {
	hg.readers.remove(hg.identifier.ID(s))
}

// Migrate re-keys the registry entry and any harvester registration from oldID to next's identity
// without stopping the harvester.
func (hg *defaultHarvesterGroup) Migrate(oldID string, next Source, updateStore func(newID string) error) error {
	newID := hg.identifier.ID(next)
	return hg.readers.migrate(oldID, newID, func() error { return updateStore(newID) })
}

// StopHarvesters stops all running Harvesters.
func (hg *defaultHarvesterGroup) StopHarvesters() error {
	return hg.tg.Stop()
}

// Lock locks a key for exclusive access and returns a resource that can be used to modify
// the cursor state and unlock the key.
func lock(ctx inputv2.Context, store *store, key string) (*resource, error) {
	resource := store.Get(key)
	if err := lockResource(ctx, resource, key); err != nil {
		return nil, err
	}
	return resource, nil
}

// lockResource locks an already retained resource for exclusive access.
func lockResource(ctx inputv2.Context, resource *resource, key string) error {
	if !resource.lock.TryLock() {
		ctx.Logger.Infof("Resource '%s' currently in use, waiting...", key)
		err := resource.lock.LockContext(ctx.Cancelation)
		ctx.Logger.Infof("Resource '%s' finally released. Lock acquired", key)
		if err != nil {
			ctx.Logger.Infof("Input for resource '%s' has been stopped while waiting", key)
			resource.Release()
			return err
		}
	}

	resource.stateMutex.Lock()
	resource.lockedVersion = resource.version
	resource.stateMutex.Unlock()

	return nil
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}

func isPermanentHarvesterError(err error) bool {
	var permanentErr permanentHarvesterError
	return errors.As(err, &permanentErr)
}
