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
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
)

// client connects a beat with the processors and pipeline queue.
//
// TODO: All ackers currently drop any late incoming ACK. Some beats still might
//       be interested in handling/waiting for event ACKs more globally
//       -> add support for not dropping pending ACKs
type client struct {
	pipeline   *Pipeline
	processors beat.Processor
	producer   queue.Producer
	mutex      sync.Mutex
	acker      beat.ACKer
	waiter     *clientCloseWaiter

	eventFlags   publisher.EventFlags
	canDrop      bool
	reportEvents bool

	// Open state, signaling, and sync primitives for coordinating client Close.
	isOpen    atomic.Bool   // set to false during shutdown, such that no new events will be accepted anymore.
	closeOnce sync.Once     // closeOnce ensure that the client shutdown sequence is only executed once
	closeRef  beat.CloseRef // extern closeRef for sending a signal that the client should be closed.
	done      chan struct{} // the done channel will be closed if the closeReg gets closed, or Close is run.

	eventer beat.ClientEventer
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
			// TODO: introduce dead-letter queue?

			log.Errorf("Failed to publish event: %v", err)
		}
	}

	if event != nil {
		e = *event
	}

	c.acker.AddEvent(e, publish)
	if !publish {
		c.onFilteredOut(e)
		return
	}

	e = *event
	pubEvent := publisher.Event{
		Content: e,
		Flags:   c.eventFlags,
	}

	if c.reportEvents {
		c.pipeline.waitCloseGroup.Add(1)
	}

	var published bool
	if c.canDrop {
		published = c.producer.TryPublish(pubEvent)
	} else {
		published = c.producer.Publish(pubEvent)
	}

	if published {
		c.onPublished()
	} else {
		c.onDroppedOnPublish(e)
		if c.reportEvents {
			c.pipeline.waitCloseGroup.Add(-1)
		}
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

		c.acker.Close()
		log.Debug("client: done closing acker")

		log.Debug("client: unlink from queue")
		c.unlink()
		log.Debug("client: done unlink")

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

// unlink is the final step of closing a client. It cancells the connect of the
// client as producer to the queue.
func (c *client) unlink() {
	log := c.logger()

	n := c.producer.Cancel() // close connection to queue
	log.Debugf("client: cancelled %v events", n)

	if c.reportEvents {
		log.Debugf("client: remove client events")
		if n > 0 {
			c.pipeline.waitCloseGroup.Add(-n)
		}
	}

	c.onClosed()
}

func (c *client) logger() *logp.Logger {
	return c.pipeline.monitors.Logger
}

func (c *client) onClosing() {
	if c.eventer != nil {
		c.eventer.Closing()
	}
}

func (c *client) onClosed() {
	c.pipeline.observer.clientClosed()
	if c.eventer != nil {
		c.eventer.Closed()
	}
}

func (c *client) onNewEvent() {
	c.pipeline.observer.newEvent()
}

func (c *client) onPublished() {
	c.pipeline.observer.publishedEvent()
	if c.eventer != nil {
		c.eventer.Published()
	}
}

func (c *client) onFilteredOut(e beat.Event) {
	log := c.logger()

	log.Debugf("Pipeline client receives callback 'onFilteredOut' for event: %+v", e)
	c.pipeline.observer.filteredEvent()
	if c.eventer != nil {
		c.eventer.FilteredOut(e)
	}
}

func (c *client) onDroppedOnPublish(e beat.Event) {
	log := c.logger()

	log.Debugf("Pipeline client receives callback 'onDroppedOnPublish' for event: %+v", e)
	c.pipeline.observer.failedPublishEvent()
	if c.eventer != nil {
		c.eventer.DroppedOnPublish(e)
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

// The Close signal from the pipeline is ignored. Instead the client
// explicitely uses `signalClose` and `wait` before it continues with the
// closing sequence.
func (w *clientCloseWaiter) Close() {}

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
