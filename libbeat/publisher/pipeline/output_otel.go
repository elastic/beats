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

//go:build !nooteloutput

package pipeline

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelconsumer"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/slabqueue"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ outputController = (*otelOutputController)(nil)

// otelOutputController is the per-pipeline outputController for a Beat
// receiver. By default the receiver path uses the slabqueue pool: every
// receiver joins a single process-global pool, keeping one in-memory event
// budget shared across all of them while keeping each receiver's FIFO
// independent (so a slow consumer on one receiver doesn't block others).
//
// The only escape hatch from slabqueue is an explicit disk queue
// configuration (queue.disk). In that case the receiver builds its own
// queue from the user-supplied factory and owns it outright.
type otelOutputController struct {
	beatInfo beat.Info
	logger   *logp.Logger
	monitors Monitors

	queue queue.Queue[publisher.Event]

	// pool is non-nil whenever the receiver uses the slabqueue pool (i.e.
	// queue.mem); it is nil when the receiver was configured with queue.disk
	// and owns its queue outright via queueFactory.
	pool *slabqueue.Pool[publisher.Event]

	// sharedPoolQueue is this controller's connection to the shared pool, used
	// to drop its budget contribution on release. Non-nil iff pool is non-nil.
	sharedPoolQueue *slabqueue.Queue[publisher.Event]

	consumer *eventConsumer

	workers    []outputWorker
	workerChan chan publisher.Batch

	// producers tracks every queue producer vended through queueProducer
	// that has not yet been closed. Each pipeline's clients normally close
	// their own producer, but on pipeline disconnection any producer still
	// open here is closed by waitClose so no producer outlives the queue it
	// publishes into. Protected by producersMu.
	producersMu sync.Mutex
	producers   map[*trackedProducer]struct{}
}

// trackedProducer wraps a queue.Producer so the owning otelOutputController
// can close any producers their clients never closed themselves when the
// pipeline disconnects. It removes itself from the controller's tracking set
// the first time Close is called — whether by the client or by the controller
// during shutdown — so Close is safe to call from both paths and only ever
// closes the underlying producer once.
//
// Publish and TryPublish are promoted from the embedded producer unchanged.
type trackedProducer struct {
	queue.Producer[publisher.Event]
	controller *otelOutputController
	closed     atomic.Bool
}

func (p *trackedProducer) Close() {
	if p.closed.Swap(true) {
		return
	}
	p.controller.untrackProducer(p)
	p.Producer.Close()
}

// otelSharedPool is the process-global slabqueue.Pool shared by every Beat
// receiver that uses the in-memory queue. A single pool gives all receivers one
// shared in-memory event budget while each keeps its own FIFO Queue, so a slow
// consumer on one receiver can't block others.
//
// Connected receivers may ask for different queue.mem.events sizes. Rather than
// rejecting a size mismatch, each receiver's own Queue is capped at its
// requested size (Queue.SetTarget), and the pool sizes itself to the largest of
// those caps — the queues drive the pool size, so the two cannot drift. The
// pool is ref-counted: created on the first connect and shut down once the last
// receiver leaves.
//
// All fields are protected by the embedded mutex; callers must hold it before
// reading or modifying either `pool` or `refs`.
var otelSharedPool = struct {
	sync.Mutex
	pool *slabqueue.Pool[publisher.Event]
	refs int
}{}

func newOTelOutputController(
	beatInfo beat.Info,
	monitors Monitors,
	retryObserver retryObserver,
	queueFactory queue.QueueFactory[publisher.Event],
	queueConfig any,
) (*otelOutputController, error) {
	var (
		pipelineQueue queue.Queue[publisher.Event]
		pool          *slabqueue.Pool[publisher.Event]
	)

	// The default receiver path uses the slabqueue pool: when queueConfig
	// is a memqueue.Settings (the default; also any explicit queue.mem),
	// we go through acquireOTelPool. Anything else (in practice
	// diskqueue.Settings from an explicit queue.disk) opts out and builds
	// its queue from the user-supplied queueFactory.
	var sharedQueue *slabqueue.Queue[publisher.Event]
	if settings, isMem := queueConfig.(memqueue.Settings); isMem {
		pool, sharedQueue = acquireOTelPool(settings, monitors)
		pipelineQueue = sharedQueue
	} else {
		// Non-memory queue (e.g. queue.disk): each receiver writes to its own
		// on-disk path, so it owns its queue outright via the user-supplied
		// factory rather than joining the shared pool.
		q, err := queueFactory(monitors.Logger, observerForMonitors(monitors), 0, nil)
		if err != nil {
			return nil, fmt.Errorf("queue creation failed: %w", err)
		}
		pipelineQueue = q
	}

	// Initialize output group.
	out, err := loadOutput(monitors, func(outStats outputs.Observer) (string, outputs.Group, error) {
		grp, err := otelconsumer.MakeOtelConsumer(beatInfo, outStats)
		return "otelconsumer", grp, err
	})
	if err != nil {
		closePipelineQueue(pipelineQueue)
		if sharedQueue != nil {
			releaseOTelPool()
		}
		return nil, err
	}

	workerChan := make(chan publisher.Batch)
	workers := make([]outputWorker, len(out.Clients))
	workerLogger := beatInfo.Logger.Named("otel_output_worker")
	for i, client := range out.Clients {
		workers[i] = makeClientWorker(workerChan, client, workerLogger, monitors.Tracer)
	}

	consumer := newEventConsumer(monitors.Logger, retryObserver)
	consumer.setTarget(consumerTarget{
		queue:      pipelineQueue,
		ch:         workerChan,
		batchSize:  out.BatchSize,
		timeToLive: out.Retry + 1,
	})

	return &otelOutputController{
		beatInfo:        beatInfo,
		logger:          beatInfo.Logger.Named("otelOutputController"),
		monitors:        monitors,
		queue:           pipelineQueue,
		pool:            pool,
		sharedPoolQueue: sharedQueue,
		consumer:        consumer,
		workers:         workers,
		workerChan:      workerChan,
		producers:       make(map[*trackedProducer]struct{}),
	}, nil
}

// closePipelineQueue force-closes the underlying queue. Used to clean up
// after a failed controller initialization.
func closePipelineQueue(q queue.Queue[publisher.Event]) {
	if q == nil {
		return
	}
	_ = q.Close(true)
}

// acquireOTelPool returns the process-global slabqueue.Pool and a fresh
// per-connection Queue into it, creating the pool on the first connect.
//
// Connections may request different event budgets. Instead of rejecting a
// mismatch, this connection's own Queue is capped at its requested size, which
// in turn sizes the shared pool to the largest cap among connected queues
// (Queue.SetTarget drives the pool). A smaller receiver therefore cannot exceed
// its own size even though the shared pool is larger, and the pool grows and
// shrinks as receivers join and leave.
func acquireOTelPool(settings memqueue.Settings, monitors Monitors) (*slabqueue.Pool[publisher.Event], *slabqueue.Queue[publisher.Event]) {
	otelSharedPool.Lock()
	defer otelSharedPool.Unlock()
	if otelSharedPool.pool == nil {
		otelSharedPool.pool = slabqueue.NewPool[publisher.Event](slabqueue.Settings{Events: settings.Events}, observerForMonitors(monitors))
		monitors.Logger.Debugf("newOTelOutputController: created shared slabqueue pool")
	}
	otelSharedPool.refs++
	q := otelSharedPool.pool.Connect()
	// Cap this connection's own queue at its requested budget. This also resizes
	// the shared pool to the largest cap among connected queues, so the pool
	// always tracks the queues and the two cannot drift.
	q.SetTarget(settings.Events)
	monitors.Logger.Debugf("newOTelOutputController: joined shared pool (%v connections, pool budget %v)", otelSharedPool.refs, otelSharedPool.pool.Target())
	return otelSharedPool.pool, q
}

// releaseOTelPool drops one reference to the shared pool and shuts it down once
// the last connected pipeline leaves. The pool's size is not adjusted here: the
// connection's queue was already closed (which resizes the pool to the
// remaining queues' caps).
func releaseOTelPool() {
	otelSharedPool.Lock()
	defer otelSharedPool.Unlock()
	if otelSharedPool.pool == nil {
		return
	}
	otelSharedPool.refs--
	if otelSharedPool.refs <= 0 {
		otelSharedPool.pool.Shutdown()
		otelSharedPool.pool = nil
		otelSharedPool.refs = 0
	}
}

// observerForMonitors returns a queue.Observer reporting metrics under the
// "pipeline.queue" registry path, matching the pre-existing layout.
func observerForMonitors(monitors Monitors) queue.Observer {
	if monitors.Metrics == nil {
		return nil
	}
	pipelineMetrics := monitors.Metrics.GetOrCreateRegistry("pipeline")
	return queue.NewQueueObserver(pipelineMetrics)
}

// waitClose disconnects this receiver pipeline. The force parameter from the
// outputController interface is ignored: a receiver always does a graceful
// drain bounded by ctx and then force-closes its own queue on timeout, so there
// is no separate force mode (a receiver must never drop a co-tenant's events).
func (c *otelOutputController) waitClose(ctx context.Context, _ bool) error {
	c.logger.Infof("Output shutdown started. Waiting for enqueued events to be published.")

	// Stop this pipeline's intake: close every producer it vended so no new
	// events enter and each producer's ACKWaitChan can resolve as its already-
	// published events are acknowledged. Clients normally close their own
	// producers first (stage one of client shutdown); this covers any that
	// did not.
	producers := c.snapshotProducers()
	for _, p := range producers {
		p.Close()
	}

	// Begin a graceful close of this pipeline's queue so the consumer keeps
	// delivering already-enqueued events and their acks fire while we wait.
	c.queue.Close(false)

	// Wait — bounded by ctx — for acknowledgments of THIS pipeline's events
	// only. Because each receiver pipeline owns its own queue and producers,
	// this never delays on another pipeline still connected to the shared pool.
	if c.waitForPipelineAcks(ctx, producers) {
		c.logger.Infof("Continue shutdown: All enqueued events have been published.")
	} else {
		// ctx expired: force-close this pipeline's queue, dropping in-flight
		// events. The queue's force-close fan-out also unblocks any producer
		// ACKWaitChan still open (see slabqueue Queue.Close).
		c.logger.Infof("Continue shutdown: Time out waiting for events to be published.")
		c.queue.Close(true)
		<-c.queue.Done()
	}

	c.consumer.close()
	close(c.workerChan)

	for _, out := range c.workers {
		out.Close()
	}

	// Idempotently close any producer vended concurrently with the snapshot
	// above, so none outlives the queue it publishes into.
	c.closeProducers()

	// Release this pipeline's claim on the shared pool. Ref-counting means only
	// the LAST connected pipeline actually shuts the pool down (Pool.Shutdown
	// force-closes any remaining queues and drains the full shared budget);
	// non-last pipelines leave the pool and other pipelines untouched. By the
	// time the last pipeline reaches here it has already waited for its own —
	// and therefore all remaining — events above. Receivers on the non-mem
	// (disk) path own their queue outright and never joined a pool.
	if c.sharedPoolQueue != nil {
		releaseOTelPool()
	}
	return nil
}

func (c *otelOutputController) queueProducer(config queue.ProducerConfig) queue.Producer[publisher.Event] {
	p := &trackedProducer{
		Producer:   c.queue.Producer(config),
		controller: c,
	}
	c.producersMu.Lock()
	c.producers[p] = struct{}{}
	c.producersMu.Unlock()
	return p
}

// untrackProducer removes a producer from the tracking set once it has been
// closed. Called from trackedProducer.Close, so it must not itself call back
// into Close.
func (c *otelOutputController) untrackProducer(p *trackedProducer) {
	c.producersMu.Lock()
	delete(c.producers, p)
	c.producersMu.Unlock()
}

// snapshotProducers returns the set of currently-tracked producers. Callers
// iterate the returned slice without holding producersMu, which is required
// because trackedProducer.Close calls back into untrackProducer (taking the
// same lock) and waiting on a producer must not hold the lock either.
func (c *otelOutputController) snapshotProducers() []*trackedProducer {
	c.producersMu.Lock()
	defer c.producersMu.Unlock()
	producers := make([]*trackedProducer, 0, len(c.producers))
	for p := range c.producers {
		producers = append(producers, p)
	}
	return producers
}

// closeProducers closes every producer that is still open when the pipeline
// disconnects.
func (c *otelOutputController) closeProducers() {
	for _, p := range c.snapshotProducers() {
		p.Close()
	}
}

// waitForPipelineAcks waits — bounded by ctx — for acknowledgments of THIS
// pipeline's events only. Each producer's ACKWaitChan closes once the producer
// is closed and its events are acknowledged; the per-pipeline queue's Done
// closes once its FIFO has fully drained. Because each receiver pipeline owns
// its own queue and producers, this never blocks on another pipeline's events.
// Returns true if everything drained, false if ctx expired first.
func (c *otelOutputController) waitForPipelineAcks(ctx context.Context, producers []*trackedProducer) bool {
	for _, p := range producers {
		select {
		case <-p.ACKWaitChan():
		case <-ctx.Done():
			return false
		}
	}
	select {
	case <-c.queue.Done():
		return true
	case <-ctx.Done():
		return false
	}
}

// poolForTest exposes the underlying pool for tests; it is not part of the
// outputController interface and must not be used outside tests.
func (c *otelOutputController) poolForTest() *slabqueue.Pool[publisher.Event] {
	return c.pool
}

// trackedProducerCountForTest reports how many vended producers are still
// open. For tests only; not part of the outputController interface.
func (c *otelOutputController) trackedProducerCountForTest() int {
	c.producersMu.Lock()
	defer c.producersMu.Unlock()
	return len(c.producers)
}
