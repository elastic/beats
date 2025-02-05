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
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// client connects a beat with the processors and pipeline queue.
type client struct {
	beatInfo beat.Info
	
	logger     *logp.Logger
	processors beat.Processor
	producer   queue.Producer
	mutex      sync.Mutex
	waiter     *clientCloseWaiter

	eventFlags publisher.EventFlags
	canDrop    bool

	// Open state, signaling, and sync primitives for coordinating client Close.
	isOpen atomic.Bool // set to false during shutdown, such that no new events will be accepted anymore.

	observer       observer
	eventListener  beat.EventListener
	clientListener beat.ClientListener
}

type clientCloseWaiter struct {
	events  atomic.Uint32
	closing atomic.Bool

	signalAll  chan struct{} // ack loop notifies `close` that all events have been acked
	signalDone chan struct{} // shutdown handler telling `wait` that shutdown has been completed
	waitClose  time.Duration
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

	c.onNewEvent(e)

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
		// TODO: double check even filtered events get a Meta, thus it can be
		//  tracked per input
		c.onFilteredOut(e)
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
		c.onPublished(e)
		// e.SetPublishStatus("published")
	} else {
		c.onDroppedOnPublish(e)
	}
}

func (c *client) Close() error {
	if c.isOpen.Swap(false) {
		// Only do shutdown handling the first time Close is called
		c.onClosing()

		c.logger.Debug("client: closing acker")
		c.waiter.signalClose()
		c.waiter.wait()

		c.eventListener.ClientClosed()
		c.logger.Debug("client: done closing acker")

		c.logger.Debug("client: close queue producer")
		c.producer.Close()
		c.onClosed()
		c.logger.Debug("client: done producer close")

		if c.processors != nil {
			c.logger.Debug("client: closing processors")
			err := processors.Close(c.processors)
			if err != nil {
				c.logger.Errorf("client: error closing processors: %v", err)
			}
			c.logger.Debug("client: done closing processors")
		}
	}
	return nil
}

func (c *client) onClosing() {
	if c.clientListener != nil {
		c.clientListener.Closing()
	}
}

func (c *client) onClosed() {
	c.observer.clientClosed()
	if c.clientListener != nil {
		c.clientListener.Closed()
	}
}

func (c *client) onNewEvent(e beat.Event) {
	c.observer.newEvent(e)
}

func (c *client) onPublished(e beat.Event) {
	c.observer.publishedEvent(e)
	if c.clientListener != nil {
		c.clientListener.Published()
	}
}

func (c *client) onFilteredOut(e beat.Event) {
	// e.SetPublishStatus("filtered")
	c.observer.filteredEvent(e)
}

func (c *client) onDroppedOnPublish(e beat.Event) {
	// e.SetPublishStatus("dropped")
	c.observer.failedPublishEvent(e)
	if c.clientListener != nil {
		c.clientListener.DroppedOnPublish(e)
	}
}

func newClientCloseWaiter(timeout time.Duration) *clientCloseWaiter {
	return &clientCloseWaiter{
		signalAll:  make(chan struct{}, 1),
		signalDone: make(chan struct{}),
		waitClose:  timeout,
	}
}

func (w *clientCloseWaiter) AddEvent(_ beat.Event, published bool) {
	if published {
		w.events.Add(1)
	}
}

func (w *clientCloseWaiter) ACKEvents(n int) {
	value := w.events.Add(^uint32(n - 1))
	if value != 0 {
		return
	}

	// send done signal, if close is waiting
	if w.closing.Load() {
		w.signalAll <- struct{}{}
	}
}

// The client's close signal is ignored. Instead the client
// explicitly uses `signalClose` and `wait` before it continues with the
// closing sequence.
func (w *clientCloseWaiter) ClientClosed() {}

func (w *clientCloseWaiter) signalClose() {
	if w == nil {
		return
	}

	w.closing.Store(true)
	if w.events.Load() == 0 {
		w.finishClose()
		return
	}

	// start routine to propagate shutdown signals or timeouts to anyone
	// being blocked in wait.
	go func() {
		defer w.finishClose()

		select {
		case <-w.signalAll:
		case <-time.After(w.waitClose):
		}
	}()
}

func (w *clientCloseWaiter) finishClose() {
	close(w.signalDone)
}

func (w *clientCloseWaiter) wait() {
	if w != nil {
		<-w.signalDone
	}
}
