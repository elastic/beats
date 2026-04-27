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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

type otelOutputController struct {
	beatInfo beat.Info
	logger   *logp.Logger
	monitors Monitors

	intakeQueueID string
	queue         queue.Queue[publisher.Event]

	// consumer is a helper goroutine that reads event batches from the queue
	// and sends them to workerChan for an output worker to process.
	consumer *eventConsumer

	// Each worker is a goroutine that will read batches from workerChan and
	// send them to the output.
	workers    []outputWorker
	workerChan chan publisher.Batch

	// The number of pipelines connected to this output controller
	pipelineCount int
}

type otelOutputControllerHandle struct {
	*otelOutputController

	beatInfo beat.Info
}

var allOutputControllers struct {
	sync.Mutex
	lookup map[string]*otelOutputController
}

func newOTelOutputController(
	beatInfo beat.Info,
	monitors Monitors,
	retryObserver retryObserver,
	queueFactory queue.QueueFactory[publisher.Event],
	intakeQueueID string,
) (*otelOutputControllerHandle, error) {
	allOutputControllers.Lock()
	defer allOutputControllers.Unlock()

	if intakeQueueID != "" {
		controller, ok := allOutputControllers.lookup[intakeQueueID]
		if ok {
			controller.pipelineCount++
			monitors.Logger.Debugf("newOTelOutputController: connecting to existing output controller for intake queue ID %v (%v pipelines connected)", intakeQueueID, controller.pipelineCount)
			return &otelOutputControllerHandle{
				otelOutputController: controller,
				beatInfo:             beatInfo,
			}, nil
		}
	} else {
		monitors.Logger.Debugf("newOTelOutputController: no intake queue ID specified")
	}

	// Queue metrics are reported under the pipeline namespace
	var pipelineMetrics *monitoring.Registry
	if monitors.Metrics != nil {
		pipelineMetrics = monitors.Metrics.GetOrCreateRegistry("pipeline")
	}
	queueObserver := queue.NewQueueObserver(pipelineMetrics)

	// Create the queue
	queue, err := queueFactory(monitors.Logger, queueObserver, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("queue creation failed: %w", err)
	}

	// Initialize output group
	out, err := loadOutput(monitors, func(outStats outputs.Observer) (string, outputs.Group, error) {
		out, err := otelconsumer.MakeOtelConsumer(beatInfo, outStats)
		return "otelconsumer", out, err
	})
	if err != nil {
		return nil, err
	}

	// Create output workers
	workerChan := make(chan publisher.Batch)
	workers := make([]outputWorker, len(out.Clients))
	logger := beatInfo.Logger.Named("otel_output_worker")
	for i, client := range out.Clients {
		workers[i] = makeClientWorker(workerChan, client, logger, monitors.Tracer)
	}

	// Create an event consumer pulling batches from the queue and sending
	// them to the output worker channel.
	consumer := newEventConsumer(monitors.Logger, retryObserver)
	consumer.setTarget(
		consumerTarget{
			queue:      queue,
			ch:         workerChan,
			batchSize:  out.BatchSize,
			timeToLive: out.Retry + 1,
		})

	controller := &otelOutputController{
		beatInfo:      beatInfo,
		logger:        beatInfo.Logger.Named("otelOutputController"),
		monitors:      monitors,
		intakeQueueID: intakeQueueID,
		queue:         queue,
		consumer:      consumer,
		workers:       workers,
		workerChan:    workerChan,
		pipelineCount: 1,
	}

	if intakeQueueID != "" {
		allOutputControllers.lookup[intakeQueueID] = controller
		monitors.Logger.Debugf("newOTelOutputController: created new output controller for intake queue ID %v", intakeQueueID)
	}

	return &otelOutputControllerHandle{
		otelOutputController: controller,
		beatInfo:             beatInfo,
	}, nil
}

func (c *otelOutputController) waitClose(ctx context.Context, _ bool) error {
	allOutputControllers.Lock()
	defer allOutputControllers.Unlock()
	if c.intakeQueueID != "" {
		c.pipelineCount--
		if c.pipelineCount > 0 {
			c.logger.Debugf("Intake queue %v: waitClose not yet supported when multiple pipelines are connected, skipping", c.intakeQueueID)
			return nil
		}
		delete(allOutputControllers.lookup, c.intakeQueueID)
	}

	// First: signal the queue that we're shutting down, and allow it to drain
	// and process ACKs until the given context terminates.
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

	// We've drained the queue as much as we can, signal eventConsumer to
	// close, and wait for it to finish. After consumer.close returns,
	// there will be no more writes to c.workerChan, so it is safe to close.
	c.consumer.close()
	close(c.workerChan)

	// Signal the output workers to close.
	for _, out := range c.workers {
		out.Close()
	}

	return nil
}

func (c *otelOutputController) queueProducer(config queue.ProducerConfig) queue.Producer[publisher.Event] {
	return c.queue.Producer(config)
}
