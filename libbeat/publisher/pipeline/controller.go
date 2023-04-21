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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	proxyqueue "github.com/elastic/beats/v7/libbeat/publisher/queue/proxy"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// outputController manages the pipelines output capabilities, like:
// - start
// - stop
// - reload
type outputController struct {
	beat     beat.Info
	monitors Monitors
	observer outputObserver

	// If eventWaitGroup is non-nil, it will be decremented as the queue
	// reports upstream acknowledgment of published events.
	eventWaitGroup *sync.WaitGroup

	// The queue is not created until the outputController is assigned a
	// nonempty outputs.Group, in case the output group requests a proxy
	// queue. At that time, any prior calls to outputController.queueProducer
	// from incoming pipeline connections will be unblocked, and future
	// requests will be handled synchronously.
	queue           queue.Queue
	queueLock       sync.Mutex
	pendingRequests []producerRequest

	// queueSettings must be one of memqueue.Settings, diskqueue.Settings, or
	// proxyqueue.Settings.
	queueSettings interface{}

	workerChan chan publisher.Batch

	consumer *eventConsumer
	workers  []outputWorker
}

type producerRequest struct {
	config       queue.ProducerConfig
	responseChan chan queue.Producer
}

// outputWorker instances pass events from the shared workQueue to the outputs.Client
// instances.
type outputWorker interface {
	Close() error
}

func newOutputController(
	beat beat.Info,
	monitors Monitors,
	observer outputObserver,
	eventWaitGroup *sync.WaitGroup,
	queueSettings interface{},
) (*outputController, error) {
	controller := &outputController{
		beat:           beat,
		monitors:       monitors,
		observer:       observer,
		eventWaitGroup: eventWaitGroup,
		queueSettings:  queueSettings,
		workerChan:     make(chan publisher.Batch),
		consumer:       newEventConsumer(monitors.Logger, observer),
	}

	return controller, nil
}

func (c *outputController) Close() error {
	c.consumer.close()
	close(c.workerChan)

	for _, out := range c.workers {
		out.Close()
	}

	// Closing the queue stops ACKs from propagating, so we close everything
	// else first to give it a chance to wait for any outstanding events to be
	// acknowledged.
	c.queue.Close()

	return nil
}

func (c *outputController) Set(outGrp outputs.Group) {
	c.createQueueIfNeeded(outGrp)

	// Set consumer to empty target to pause it while we reload
	c.consumer.setTarget(consumerTarget{})

	// Close old outputWorkers, so they send their remaining events
	// back to eventConsumer's retry channel
	for _, w := range c.workers {
		w.Close()
	}

	// create new output group with the shared work queue
	clients := outGrp.Clients
	c.workers = make([]outputWorker, len(clients))
	for i, client := range clients {
		logger := logp.NewLogger("publisher_pipeline_output")
		c.workers[i] = makeClientWorker(c.workerChan, client, logger, c.monitors.Tracer)
	}

	targetChan := c.workerChan
	if len(clients) == 0 {
		// If there are no output clients, we are probably still waiting
		// for our output config from Agent via BeatV2Manager.reloadOutput.
		// In this case outGrp.BatchSize is probably 0, allowing arbitrarily
		// large batches. Set the work channel to nil so eventConsumer
		// doesn't prime the pipeline with such batches until we get the
		// requested batch size for the real output.
		targetChan = nil
	}

	// Resume consumer targeting the new work queue
	c.consumer.setTarget(
		consumerTarget{
			queue:      c.queue,
			ch:         targetChan,
			batchSize:  outGrp.BatchSize,
			timeToLive: outGrp.Retry + 1,
		})
}

// Reload the output
func (c *outputController) Reload(
	cfg *reload.ConfigWithMeta,
	outFactory func(outputs.Observer, conf.Namespace) (outputs.Group, error),
) error {
	outCfg := conf.Namespace{}
	if cfg != nil {
		if err := cfg.Config.Unpack(&outCfg); err != nil {
			return err
		}
	}

	output, err := loadOutput(c.monitors, func(stats outputs.Observer) (string, outputs.Group, error) {
		name := outCfg.Name()
		out, err := outFactory(stats, outCfg)
		return name, out, err
	})
	if err != nil {
		return err
	}

	c.Set(output)

	return nil
}

// queueProducer creates a queue producer with the given config, blocking
// until the queue is created if it does not yet exist.
func (c *outputController) queueProducer(config queue.ProducerConfig) queue.Producer {
	c.queueLock.Lock()
	if c.queue != nil {
		// We defer the unlock only after the nil check because if the
		// queue doesn't exist we'll need to block until it does, and
		// in that case we need to manually unlock before we start waiting.
		defer c.queueLock.Unlock()
		return c.queue.Producer(config)
	}
	// If there's no queue yet, create a producer request, release the
	// queue lock, and wait to receive our producer.
	request := producerRequest{
		config:       config,
		responseChan: make(chan queue.Producer),
	}
	c.pendingRequests = append(c.pendingRequests, request)
	c.queueLock.Unlock()
	return <-request.responseChan
}

// onACK receives event acknowledgment notifications from the queue and
// forwards them to the metrics observer and the pipeline's global event
// wait group if one is set.
func (c *outputController) onACK(eventCount int) {
	c.observer.queueACKed(eventCount)
	if c.eventWaitGroup != nil {
		c.eventWaitGroup.Add(-eventCount)
	}
}

func (c *outputController) createQueueIfNeeded(outGrp outputs.Group) {
	logger := c.monitors.Logger
	if len(outGrp.Clients) == 0 {
		// If the output group is empty, there's nothing to do
		return
	}
	c.queueLock.Lock()
	defer c.queueLock.Unlock()

	if c.queue != nil {
		// Some day we might support hot-swapping of output configurations,
		// but for now we can only accept a nonempty output group once, and
		// after that we log it as an error.
		logger.Errorf("outputController received new output configuration when queue is already active")
		return
	}

	// Queue settings from the output take precedence, otherwise fall back
	// on what we were given during initialization.
	settings := outGrp.QueueSettings
	if settings == nil {
		settings = c.queueSettings
	}

	var queueType string
	switch s := settings.(type) {
	case proxyqueue.Settings:
		queueType = proxyqueue.QueueType
		c.queue = proxyqueue.NewQueue(logger, c.onACK, s)

	case memqueue.Settings:
		queueType = memqueue.QueueType
		c.queue = memqueue.NewQueue(logger, c.onACK, s)

	case diskqueue.Settings:
		queueType = diskqueue.QueueType
		queue, err := diskqueue.NewQueue(logger, c.onACK, s)
		if err != nil {
			logger.Errorf("couldn't initialize disk queue: %v", err)
		}
		c.queue = queue
	default:
		logger.Errorf("outputController received unrecognized queue settings")
	}
	// If we have no valid settings or the disk queue failed, fall back on the
	// memory queue since that is guaranteed to succeed.
	if c.queue == nil {
		queueType = memqueue.QueueType
		logger.Errorf("falling back to default memory queue, check your queue configuration")
		s, _ := memqueue.SettingsForUserConfig(nil)
		c.queue = memqueue.NewQueue(logger, c.onACK, s)
	}

	if c.monitors.Telemetry != nil {
		queueReg := c.monitors.Telemetry.NewRegistry("queue")
		monitoring.NewString(queueReg, "name").Set(queueType)
	}
	maxEvents := c.queue.BufferConfig().MaxEvents
	c.observer.queueMaxEvents(maxEvents)

	// Now that we've created a queue, go through and unblock any callers
	// that are waiting for a producer.
	for _, req := range c.pendingRequests {
		req.responseChan <- c.queue.Producer(req.config)
	}
	c.pendingRequests = nil
}
