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
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelconsumer"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/otelqueue"
	"github.com/elastic/elastic-agent-libs/logp"
)

var _ outputController = (*otelOutputController)(nil)

// otelOutputController is the per-pipeline outputController for a Beat
// receiver. Each connected pipeline owns its own queue (a façade over a
// shared otelqueue.Pool), its own event consumer, and its own set of output
// workers wired to its own LogConsumer.
//
// Multiple pipelines that share an intake queue ID share the underlying
// otelqueue.Pool. The pool is the shared event budget; per-pipeline queues
// keep events isolated, so a slow consumer on one pipeline cannot block
// other pipelines' deliveries.
type otelOutputController struct {
	beatInfo beat.Info
	logger   *logp.Logger
	monitors Monitors

	intakeQueueID string
	queue         queue.Queue[publisher.Event]
	pool          *otelqueue.Pool[publisher.Event]

	consumer *eventConsumer

	workers    []outputWorker
	workerChan chan publisher.Batch
}

// sharedPool tracks one otelqueue.Pool together with its connected pipeline
// count, for ref-counted lifecycle of pools indexed by intake queue ID.
type sharedPool struct {
	pool     *otelqueue.Pool[publisher.Event]
	settings memqueue.Settings // remembered so later joiners can be validated
	refs     int
}

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
	queueConfig any,
) (*otelOutputController, error) {
	// publisher.Event.Source-style routing is no longer used by this
	// implementation, but we still require an in-memory queue configuration
	// because the receiver path does not support a disk-backed pool.
	settings, ok := queueConfig.(memqueue.Settings)
	if !ok {
		return nil, fmt.Errorf("receiver output requires an in-memory queue configuration, got %T", queueConfig)
	}

	// Get (or create) the pool for this intake queue ID. An empty ID means
	// "private pool for this pipeline only" and is never registered.
	pool, err := acquireOTelPool(intakeQueueID, settings, monitors)
	if err != nil {
		return nil, err
	}

	pipelineQueue := pool.Connect()

	// Initialize output group.
	out, err := loadOutput(monitors, func(outStats outputs.Observer) (string, outputs.Group, error) {
		grp, err := otelconsumer.MakeOtelConsumer(beatInfo, outStats)
		return "otelconsumer", grp, err
	})
	if err != nil {
		pipelineQueue.Close(true)
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

// acquireOTelPool returns the otelqueue.Pool for the given intake queue ID,
// creating it if necessary. An empty ID returns a private pool that is not
// shared and not tracked in the global registry. Settings mismatches against
// an already-registered ID return an error.
func acquireOTelPool(intakeQueueID string, settings memqueue.Settings, monitors Monitors) (*otelqueue.Pool[publisher.Event], error) {
	poolSettings := otelqueue.Settings{Events: settings.Events}

	if intakeQueueID == "" {
		return otelqueue.NewPool[publisher.Event](poolSettings, observerForMonitors(monitors)), nil
	}

	allOTelPools.Lock()
	defer allOTelPools.Unlock()
	if existing, ok := allOTelPools.lookup[intakeQueueID]; ok {
		if existing.settings.Events != settings.Events {
			return nil, fmt.Errorf("shared intake queue %q already initialized with different queue settings", intakeQueueID)
		}
		existing.refs++
		monitors.Logger.Debugf("newOTelOutputController: joining existing pool for intake queue ID %v (%v pipelines connected)", intakeQueueID, existing.refs)
		return existing.pool, nil
	}
	pool := otelqueue.NewPool[publisher.Event](poolSettings, observerForMonitors(monitors))
	allOTelPools.lookup[intakeQueueID] = &sharedPool{pool: pool, settings: settings, refs: 1}
	monitors.Logger.Debugf("newOTelOutputController: created new pool for intake queue ID %v", intakeQueueID)
	return pool, nil
}

// releaseOTelPool decrements the ref count for the given intake queue ID and
// shuts the pool down once the last connected pipeline leaves.
func releaseOTelPool(intakeQueueID string) {
	if intakeQueueID == "" {
		return
	}
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
func (c *otelOutputController) poolForTest() *otelqueue.Pool[publisher.Event] {
	return c.pool
}
