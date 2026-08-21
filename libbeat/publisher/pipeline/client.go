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
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ beat.Client = (*client)(nil)

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
	mutex      sync.Mutex

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

func (c *client) publish(e beat.Event) {
	var (
		event   = &e
		publish = true
	)

	c.onNewEvent()

	if !c.isOpen.Load() {
		// client is closing down -> report event as dropped and return
		c.onDroppedOnPublish(e)
		return
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
		e = *event
	}

	c.eventListener.AddEvent(e, publish)
	if !publish {
		c.onFilteredOut()
		return
	}

	e = *event
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
