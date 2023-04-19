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
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
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

	queueConfig queueConfig
	queue       queue.Queue

	workerChan chan publisher.Batch

	consumer *eventConsumer
	workers  []outputWorker
}

// outputWorker instances pass events from the shared workQueue to the outputs.Client
// instances.
type outputWorker interface {
	Close() error
}

type queueConfig struct {
	logger      *logp.Logger
	queueType   string
	userConfig  *conf.C
	ackCallback func(eventCount int)
	inQueueSize int
}

func newOutputController(
	beat beat.Info,
	monitors Monitors,
	observer outputObserver,
	queueConfig queueConfig,
) (*outputController, error) {

	controller := &outputController{
		beat:        beat,
		monitors:    monitors,
		observer:    observer,
		queueConfig: queueConfig,
		workerChan:  make(chan publisher.Batch),
		consumer:    newEventConsumer(monitors.Logger, observer),
	}

	err := controller.createQueue()
	if err != nil {
		return nil, err
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

func (c *outputController) queueProducer(config queue.ProducerConfig) queue.Producer {
	return c.queue.Producer(config)
}

func (c *outputController) createQueue() error {
	config := c.queueConfig

	switch config.queueType {
	case memqueue.QueueType:
		settings, err := memqueue.SettingsForUserConfig(config.userConfig)
		if err != nil {
			return err
		}
		// The memory queue has a special override during pipeline
		// initialization for the size of its API channel buffer.
		settings.InputQueueSize = config.inQueueSize
		settings.ACKCallback = config.ackCallback
		c.queue = memqueue.NewQueue(config.logger, settings)
	case diskqueue.QueueType:
		settings, err := diskqueue.SettingsForUserConfig(config.userConfig)
		if err != nil {
			return err
		}
		settings.WriteToDiskCallback = config.ackCallback
		queue, err := diskqueue.NewQueue(config.logger, settings)
		if err != nil {
			return err
		}
		c.queue = queue
	default:
		return fmt.Errorf("'%v' is not a valid queue type", config.queueType)
	}

	if c.monitors.Telemetry != nil {
		queueReg := c.monitors.Telemetry.NewRegistry("queue")
		monitoring.NewString(queueReg, "name").Set(config.queueType)
	}
	maxEvents := c.queue.BufferConfig().MaxEvents
	c.observer.queueMaxEvents(maxEvents)

	return nil
}
