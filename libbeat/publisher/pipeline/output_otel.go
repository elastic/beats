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
// receiver. By default the receiver path uses the slabqueue pool: an
// explicit intake queue ID joins the pool registered under that name, and
// no ID joins the global default pool. Either way the pool is shared
// across every receiver with the same ID, keeping a single in-memory event
// budget across all of them while keeping each receiver's FIFO independent
// (so a slow consumer on one receiver doesn't block others).
//
// The only escape hatch from slabqueue is an explicit disk queue
// configuration (queue.disk). In that case the receiver builds its own
// queue from the user-supplied factory — sharing is not supported for disk
// queues, so an intake queue ID combined with queue.disk is rejected.
type otelOutputController struct {
	beatInfo beat.Info
	logger   *logp.Logger
	monitors Monitors

	intakeQueueID string
	queue         queue.Queue[publisher.Event]
	// pool is non-nil whenever the receiver uses the slabqueue pool (i.e.
	// queue.mem); it is nil when the receiver was configured with queue.disk
	// and owns its queue outright via queueFactory.
	pool *slabqueue.Pool[publisher.Event]

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

// sharedPool tracks one slabqueue.Pool together with its connected
// pipeline count, for ref-counted lifecycle of pools indexed by intake
// queue ID.
//
// All fields are protected by allOTelPools.Mutex — sharedPool itself
// carries no lock, and instances live only inside allOTelPools.lookup,
// so a sharedPool value should never be touched outside a section that
// holds the registry lock.
type sharedPool struct {
	pool     *slabqueue.Pool[publisher.Event]
	settings memqueue.Settings // remembered so later joiners can be validated
	refs     int
}

// allOTelPools is the process-global registry of slabqueue.Pools keyed
// by intake queue ID. The embedded mutex protects both `lookup` and every
// field of every *sharedPool it contains; callers must always acquire it
// before reading or modifying either.
var allOTelPools = struct {
	sync.Mutex
	lookup map[string]*sharedPool
}{
	lookup: make(map[string]*sharedPool),
}

func newOTelOutputController(
	beatInfo beat.Info,
	monitors Monitors,
	retryObserver retryObserver,
	intakeQueueID string,
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
	if settings, isMem := queueConfig.(memqueue.Settings); isMem {
		var err error
		pool, err = acquireOTelPool(intakeQueueID, settings, monitors)
		if err != nil {
			return nil, err
		}
		pipelineQueue = pool.Connect()
	} else {
		// Non-memory queue (e.g. queue.disk): sharing is meaningless because
		// each receiver writes to its own on-disk path, so an intake queue
		// ID combined with a non-memory queue is a configuration error.
		if intakeQueueID != "" {
			return nil, fmt.Errorf("shared intake queue %q requires queue.mem, got %T", intakeQueueID, queueConfig)
		}
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
		releaseOTelPool(intakeQueueID)
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
		beatInfo:      beatInfo,
		logger:        beatInfo.Logger.Named("otelOutputController"),
		monitors:      monitors,
		intakeQueueID: intakeQueueID,
		queue:         pipelineQueue,
		pool:          pool,
		consumer:      consumer,
		workers:       workers,
		workerChan:    workerChan,
		producers:     make(map[*trackedProducer]struct{}),
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

// acquireOTelPool returns the slabqueue.Pool for the given intake queue ID,
// creating it if necessary. The empty string is the global default ID:
// every receiver started without an explicit intake queue ID joins the
// same global pool. Settings mismatches against an already-registered ID
// (including the global one) return an error.
func acquireOTelPool(intakeQueueID string, settings memqueue.Settings, monitors Monitors) (*slabqueue.Pool[publisher.Event], error) {
	poolSettings := slabqueue.Settings{Events: settings.Events}

	allOTelPools.Lock()
	defer allOTelPools.Unlock()
	if existing, ok := allOTelPools.lookup[intakeQueueID]; ok {
		if existing.settings.Events != settings.Events {
			return nil, fmt.Errorf("shared intake queue %q already initialized with different queue settings", intakeQueueID)
		}
		existing.refs++
		monitors.Logger.Debugf("newOTelOutputController: joining existing pool for intake queue ID %q (%v pipelines connected)", intakeQueueID, existing.refs)
		return existing.pool, nil
	}
	pool := slabqueue.NewPool[publisher.Event](poolSettings, observerForMonitors(monitors))
	allOTelPools.lookup[intakeQueueID] = &sharedPool{pool: pool, settings: settings, refs: 1}
	monitors.Logger.Debugf("newOTelOutputController: created new pool for intake queue ID %q", intakeQueueID)
	return pool, nil
}

// releaseOTelPool decrements the ref count for the given intake queue ID and
// shuts the pool down once the last connected pipeline leaves. The empty ID
// is the global default pool and is treated the same as a named ID.
func releaseOTelPool(intakeQueueID string) {
	allOTelPools.Lock()
	defer allOTelPools.Unlock()
	entry, ok := allOTelPools.lookup[intakeQueueID]
	if !ok {
		return
	}
	entry.refs--
	if entry.refs == 0 {
		entry.pool.Shutdown()
		delete(allOTelPools.lookup, intakeQueueID)
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
	// and therefore all remaining — events above.
	releaseOTelPool(c.intakeQueueID)
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
