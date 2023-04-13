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
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/logp"
)

// client connects a beat with the processors and pipeline queue.
type client struct {
	pipeline   *Pipeline
	processors beat.Processor
	producer   queue.Producer
	mutex      sync.Mutex
	waiter     *clientCloseWaiter

	eventFlags   publisher.EventFlags
	canDrop      bool
	reportEvents bool

	// Open state, signaling, and sync primitives for coordinating client Close.
	isOpen    atomic.Bool   // set to false during shutdown, such that no new events will be accepted anymore.
	closeOnce sync.Once     // closeOnce ensure that the client shutdown sequence is only executed once
	closeRef  beat.CloseRef // extern closeRef for sending a signal that the client should be closed.
	done      chan struct{} // the done channel will be closed if the closeReg gets closed, or Close is run.

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
		log     = c.pipeline.monitors.Logger
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
			log.Errorf("Failed to publish event: %v", err)
		}
	}

	if event != nil {
		e = *event
	}

	c.eventListener.AddEvent(e, publish)
	if !publish {
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
		c.onPublished()
	} else {
		c.onDroppedOnPublish(e)
	}
}

func (c *client) Close() error {
	log := c.logger()

	// first stop ack handling. ACK handler might block on wait (with timeout), waiting
	// for pending events to be ACKed.
	c.closeOnce.Do(func() {
		close(c.done)

		c.isOpen.Store(false)
		c.onClosing()

		log.Debug("client: closing acker")
		c.waiter.signalClose()
		c.waiter.wait()

		c.eventListener.ClientClosed()
		log.Debug("client: done closing acker")

		log.Debug("client: close queue producer")
		cancelledEventCount := c.producer.Cancel()
		c.onClosed(cancelledEventCount)
		log.Debug("client: done producer close")

		if c.processors != nil {
			log.Debug("client: closing processors")
			err := processors.Close(c.processors)
			if err != nil {
				log.Errorf("client: error closing processors: %v", err)
			}
			log.Debug("client: done closing processors")
		}
	})
	return nil
}

func (c *client) logger() *logp.Logger {
	return c.pipeline.monitors.Logger
}

func (c *client) onClosing() {
	if c.clientListener != nil {
		c.clientListener.Closing()
	}
}

func (c *client) onClosed(cancelledEventCount int) {
	log := c.logger()
	log.Debugf("client: cancelled %v events", cancelledEventCount)

	if c.reportEvents {
		log.Debugf("client: remove client events")
		if cancelledEventCount > 0 {
			c.pipeline.waitCloseGroup.Add(-cancelledEventCount)
		}
	}

	c.pipeline.observer.clientClosed()
	if c.clientListener != nil {
		c.clientListener.Closed()
	}
}

func (c *client) onNewEvent() {
	c.pipeline.observer.newEvent()
}

func (c *client) onPublished() {
	if c.reportEvents {
		c.pipeline.waitCloseGroup.Add(1)
	}
	c.pipeline.observer.publishedEvent()
	if c.clientListener != nil {
		c.clientListener.Published()
	}
}

func (c *client) onFilteredOut(e beat.Event) {
	log := c.logger()

	log.Debugf("Pipeline client receives callback 'onFilteredOut' for event: %+v", e)
	c.pipeline.observer.filteredEvent()
	if c.clientListener != nil {
		c.clientListener.FilteredOut(e)
	}
}

func (c *client) onDroppedOnPublish(e beat.Event) {
	log := c.logger()

	log.Debugf("Pipeline client receives callback 'onDroppedOnPublish' for event: %+v", e)
	c.pipeline.observer.failedPublishEvent()
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
		w.events.Inc()
	}
}

func (w *clientCloseWaiter) ACKEvents(n int) {
	value := w.events.Sub(uint32(n))
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
