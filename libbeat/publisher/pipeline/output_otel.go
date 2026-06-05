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

func (c *otelOutputController) waitClose(ctx context.Context, _ bool) error {
	c.logger.Infof("Output shutdown started. Waiting for enqueued events to be published.")
	c.queue.Close(false)
	select {
	case <-c.queue.Done():
		c.logger.Infof("Continue shutdown: All enqueued events have been published.")
	case <-ctx.Done():
		c.logger.Infof("Continue shutdown: Time out waiting for events to be published.")
		c.queue.Close(true)
		<-c.queue.Done()
	}

	c.consumer.close()
	close(c.workerChan)

	for _, out := range c.workers {
		out.Close()
	}

	// Release this pipeline's claim on the shared pool. When the last
	// connected pipeline releases, the pool is shut down.
	releaseOTelPool(c.intakeQueueID)
	return nil
}

func (c *otelOutputController) queueProducer(config queue.ProducerConfig) queue.Producer[publisher.Event] {
	return c.queue.Producer(config)
}

// poolForTest exposes the underlying pool for tests; it is not part of the
// outputController interface and must not be used outside tests.
func (c *otelOutputController) poolForTest() *slabqueue.Pool[publisher.Event] {
	return c.pool
}
