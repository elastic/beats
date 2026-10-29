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

package pipeline

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	_ beat.Client           = (*client)(nil)
	_ beat.ContextPublisher = (*client)(nil)
)

// client implements beat.Client interface
// client connects a beat with the processors and pipeline queue.
//
// Shutdown is two-stage. Close (stage one, called by the client's owner) stops
// accepting new events and closes the queue producer, then returns immediately;
// acknowledgments for already-published events keep flowing. disconnect (stage
// two, called only by the owning Pipeline) stops accepting acknowledgments and
// drops all references to the client. Splitting the two lets a Beater close its
// inputs without blocking, while the Pipeline owns when acknowledgments are
// finalized — see issues #50104 and #49794.
type client struct {
	logger     *logp.Logger
	processors beat.Processor
	producer   queue.Producer[publisher.Event]
	// ctxProducer is producer when PublishContext may cancel a publish to it,
	// else nil (see cancellableProducer).
	ctxProducer queue.ContextProducer[publisher.Event]
	// closeCtx is cancelled by Close to release a PublishContext waiting for
	// room in the queue. Set only when ctxProducer is non-nil.
	closeCtx    context.Context
	closeCancel context.CancelFunc
	mutex       sync.Mutex

	// merged is the last context built by publishCtx, for a caller ctx whose
	// Done channel is mergedDone.
	merged       context.Context
	mergedDone   <-chan struct{}
	mergedCancel func()

	eventFlags publisher.EventFlags
	canDrop    bool

	// Open state, signaling, and sync primitives for coordinating client Close.
	isOpen       atomic.Bool // set to false during shutdown, such that no new events will be accepted anymore.
	disconnected atomic.Bool // set the first time disconnect runs, so the second stage is idempotent.

	// onRemove, if set, unregisters this client from its owning Pipeline. It is
	// run once, from disconnect.
	onRemove func()

	// requestFinalize, if set, hands this client to the owning Pipeline's reaper after
	// Close so it is finalized (stage two) as soon as its events drain, instead
	// of lingering until the whole pipeline disconnects. Run once, from Close.
	requestFinalize func()

	observer       observer
	eventListener  beat.EventListener
	clientListener beat.ClientListener
}

func (c *client) PublishAll(events []beat.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, e := range events {
		c.publish(e)
	}
}

func (c *client) Publish(e beat.Event) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.publish(e)
}

// publish publishes e and reports whether it was queued or filtered out; false
// means the event was dropped.
func (c *client) publish(e beat.Event) bool {
	if !c.prepare(&e) {
		return c.isOpen.Load()
	}

	pubEvent := publisher.Event{
		Content: e,
		Flags:   c.eventFlags,
	}

	var published bool
	if c.canDrop {
		_, published = c.producer.TryPublish(pubEvent)
	} else {
		_, published = c.producer.Publish(pubEvent)
	}

	if published {
		c.onPublished()
	} else {
		c.onDroppedOnPublish(e)
	}
	return published
}

// PublishContext implements beat.ContextPublisher. When the publish cannot be
// cancelled (see cancellableProducer), or the client drops events on a full
// queue, it publishes like Publish.
func (c *client) PublishContext(ctx context.Context, e beat.Event) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.canDrop || c.ctxProducer == nil {
		switch {
		case c.publish(e):
			return nil
		case !c.isOpen.Load():
			return beat.ErrClientClosed
		case c.canDrop:
			// With DropIfFull a refused event may only mean a full queue,
			// which is not an error.
			return nil
		default:
			return beat.ErrRejected
		}
	}

	if !c.prepare(&e) {
		if !c.isOpen.Load() {
			return beat.ErrClientClosed
		}
		return nil // filtered out
	}

	_, err := c.ctxProducer.PublishContext(c.publishCtx(ctx), publisher.Event{
		Content: e,
		Flags:   c.eventFlags,
	})
	if err != nil {
		c.onDroppedOnPublish(e)
		switch {
		case errors.Is(err, queue.ErrClosed) || c.closeCtx.Err() != nil:
			return beat.ErrClientClosed
		case errors.Is(err, errRejected):
			return beat.ErrRejected
		}
		return err
	}
	c.onPublished()
	return nil
}

// errRejected is returned by a ContextProducer wrapping a producer that cannot
// cancel (trackedProducer around the disk queue's) when Publish refuses an
// event although the wrapper is open. PublishContext reports it as
// beat.ErrRejected unless the client is closing.
var errRejected = errors.New("pipeline: event rejected by queue")

// publishCtx returns a context that is done when ctx is done or the client
// closes. Callers usually pass the same ctx many times, so the merged context
// is kept and keyed on ctx's Done channel. A ctx with the same Done channel,
// such as a context.WithValue child, reuses it: its values are not visible
// through the merged context, but a producer only uses Done and Err. c.mutex
// must be held.
func (c *client) publishCtx(ctx context.Context) context.Context {
	done := ctx.Done()
	if done == nil {
		return c.closeCtx // ctx is never done
	}
	if done != c.mergedDone {
		if c.mergedCancel != nil {
			c.mergedCancel()
		}
		merged, cancel := context.WithCancel(ctx)
		stop := context.AfterFunc(c.closeCtx, cancel)
		c.merged, c.mergedDone = merged, done
		c.mergedCancel = func() {
			stop()
			cancel()
		}
	}
	return c.merged
}

// cancellableProducer returns p as a queue.ContextProducer if a publish to it
// may be cancelled, else nil. A client with an event listener cannot cancel:
// the listener has already seen the event in AddEvent, and listeners that
// count events in order (acker.TrackingCounter, EventPrivateReporter) would
// then wait for an ACK that never comes.
func cancellableProducer(p queue.Producer[publisher.Event], l beat.EventListener) queue.ContextProducer[publisher.Event] {
	if l != nil {
		return nil
	}
	cp, _ := p.(queue.ContextProducer[publisher.Event])
	return cp
}

// prepare does the bookkeeping and processing shared by every publish path,
// leaving the processed event in e. It returns false, having reported the
// event, if the client is closed or the processors filtered the event out.
func (c *client) prepare(e *beat.Event) bool {
	var (
		event   = e
		publish = true
	)

	c.onNewEvent()

	if !c.isOpen.Load() {
		// client is closing down -> report event as dropped and return
		c.onDroppedOnPublish(*e)
		return false
	}

	if c.processors != nil {
		var err error

		event, err = c.processors.Run(event)
		publish = event != nil
		if err != nil {
			// If we introduce a dead-letter queue, this is where we should
			// route the event to it.
			c.logger.Errorf("Failed to publish event: %v", err)
		}
	}

	if event != nil {
		*e = *event
	}

	c.eventListener.AddEvent(*e, publish)
	if !publish {
		c.onFilteredOut()
		return false
	}
	return true
}

// Close performs stage one of shutdown: it stops the client from accepting new
// events and closes the underlying queue producer, then returns immediately. It
// does NOT wait for acknowledgments — acks for already-published events keep
// flowing through the event listener until the owning Pipeline calls disconnect.
//
// Note: unlike before, Close no longer blocks for ClientConfig.WaitClose. The
// pipeline-level shutdown (Pipeline.Disconnect, bounded by its context) is now
// responsible for waiting on outstanding acknowledgments.
func (c *client) Close() error {
	if c.closeCancel != nil {
		// Release a PublishContext waiting for room in the queue: it holds
		// the mutex.
		c.closeCancel()
	}
	// Hold the mutex so any in-progress Publish finishes before we flip isOpen.
	c.mutex.Lock()
	if !c.isOpen.Swap(false) {
		c.mutex.Unlock()
		return nil
	}
	c.onClosing()
	c.mutex.Unlock()

	c.logger.Debug("client: close queue producer")
	c.producer.Close()
	c.logger.Debug("client: done producer close")

	// Processors only run on the publish path, which is now closed, so it is
	// safe to release them here rather than deferring to disconnect.
	if c.processors != nil {
		c.logger.Debug("client: closing processors")
		err := processors.Close(c.processors)
		if err != nil {
			c.logger.Errorf("client: error closing processors: %v", err)
		}
		c.logger.Debug("client: done closing processors")
	}

	// Hand off to the pipeline reaper to finalize (stage two) once this
	// client's already-published events are acknowledged. The Pipeline also
	// finalizes any still-registered client on Disconnect, so this is a
	// best-effort early cleanup.
	if c.requestFinalize != nil {
		c.requestFinalize()
	}
	return nil
}

// disconnect performs stage two of shutdown: it stops accepting acknowledgments
// and drops all references to the client so a restarting pipeline cannot collide
// with it or leak it. It is invoked exactly once by the owning Pipeline (never
// by user code) and is idempotent.
func (c *client) disconnect() {
	if c.disconnected.Swap(true) {
		return
	}
	c.eventListener.ClientClosed()
	c.logger.Debug("client: done closing acker")
	c.onClosed()
	if c.onRemove != nil {
		c.onRemove()
	}
}

func (c *client) onClosing() {
	c.clientListener.Closing()
}

func (c *client) onClosed() {
	c.observer.clientClosed()
	c.clientListener.Closed()
}

func (c *client) onNewEvent() {
	c.observer.newEvent()
	c.clientListener.NewEvent()
}

func (c *client) onPublished() {
	c.observer.publishedEvent()
	c.clientListener.Published()
}

func (c *client) onFilteredOut() {
	c.observer.filteredEvent()
	c.clientListener.Filtered()
}

func (c *client) onDroppedOnPublish(e beat.Event) {
	c.observer.failedPublishEvent()
	c.clientListener.DroppedOnPublish(e)
}
