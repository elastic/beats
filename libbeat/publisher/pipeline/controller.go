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
	"github.com/elastic/elastic-agent-libs/config"
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

	queue queue.Queue

	workerChan chan publisher.Batch

	consumer *eventConsumer
	workers  []outputWorker
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
	queueConfig QueueConfig,
	ackCallback func(eventCount int),
	inQueueSize int,
) (*outputController, error) {

	controller := &outputController{
		beat:       beat,
		monitors:   monitors,
		observer:   observer,
		workerChan: make(chan publisher.Batch),
		consumer:   newEventConsumer(monitors.Logger, observer),
	}

	queueFactory, err := createQueueFactory(queueConfig, monitors, inQueueSize)
	if err != nil {
		return nil, err
	}
	queue, err := queueFactory(ackCallback)
	controller.queue = queue

	return controller, nil
}

func (c *outputController) Close() error {
	c.consumer.close()
	close(c.workerChan)

	for _, out := range c.workers {
		out.Close()
	}
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
		c.workers[i] = makeClientWorker(c.observer, c.workerChan, client, logger, c.monitors.Tracer)
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
			ch:         targetChan,
			batchSize:  outGrp.BatchSize,
			timeToLive: outGrp.Retry + 1,
		})
}

// Reload the output
func (c *outputController) Reload(
	cfg *reload.ConfigWithMeta,
	outFactory func(outputs.Observer, config.Namespace) (outputs.Group, error),
) error {
	outCfg := config.Namespace{}
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
	return nil
}

func createQueueFactory(
	config QueueConfig,
	monitors Monitors,
	inQueueSize int,
) (queueFactory, error) {
	if monitors.Telemetry != nil {
		queueReg := monitors.Telemetry.NewRegistry("queue")
		monitoring.NewString(queueReg, "name").Set(config.Type)
	}

	switch config.Type {
	case memqueue.QueueType:
		settings, err := memqueue.SettingsForUserConfig(config.UserConfig)
		if err != nil {
			return nil, err
		}
		// The memory queue has a special override during pipeline
		// initialization for the size of its API channel buffer.
		settings.InputQueueSize = inQueueSize
		return memQueueFactory(monitors.Logger, settings), nil
	case diskqueue.QueueType:
		settings, err := diskqueue.SettingsForUserConfig(config.UserConfig)
		if err != nil {
			return nil, err
		}
		return diskQueueFactory(monitors.Logger, settings), nil
	default:
		return nil, fmt.Errorf("'%v' is not a valid queue type", config.Type)
	}
}

func memQueueFactory(logger *logp.Logger, settings memqueue.Settings) queueFactory {
	return func(ackCallback func(eventCount int)) (queue.Queue, error) {
		factorySettings := settings
		factorySettings.ACKCallback = ackCallback
		return memqueue.NewQueue(logger, factorySettings), nil
	}
}

func diskQueueFactory(logger *logp.Logger, settings diskqueue.Settings) queueFactory {
	return func(ackCallback func(eventCount int)) (queue.Queue, error) {
		factorySettings := settings
		factorySettings.WriteToDiskCallback = ackCallback
		return diskqueue.NewQueue(logger, factorySettings)
	}
}
