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
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/reload"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/outputs"
	"github.com/menderesk/beats/v7/libbeat/publisher"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
)

// outputController manages the pipelines output capabilities, like:
// - start
// - stop
// - reload
type outputController struct {
	beat     beat.Info
	monitors Monitors
	observer outputObserver

	workQueue chan publisher.Batch

	consumer *eventConsumer
	out      *outputGroup
}

// outputGroup configures a group of load balanced outputs with shared work queue.
type outputGroup struct {
	// workQueue is a channel that receives event batches that
	// are ready to send. Each output worker in outputs reads from
	// workQueue for events to send.
	workQueue chan publisher.Batch
	outputs   []outputWorker

	batchSize  int
	timeToLive int // event lifetime
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
	queue queue.Queue,
) *outputController {
	return &outputController{
		beat:      beat,
		monitors:  monitors,
		observer:  observer,
		workQueue: make(chan publisher.Batch),
		consumer:  newEventConsumer(monitors.Logger, queue, observer),
	}
}

func (c *outputController) Close() error {
	c.consumer.close()
	close(c.workQueue)

	if c.out != nil {
		for _, out := range c.out.outputs {
			out.Close()
		}
	}
	return nil
}

func (c *outputController) Set(outGrp outputs.Group) {
	// Set consumer to empty target to pause it while we reload
	c.consumer.setTarget(consumerTarget{})

	// Close old outputWorkers, so they send their remaining events
	// back to eventConsumer's retry channel
	if c.out != nil {
		for _, w := range c.out.outputs {
			w.Close()
		}
	}

	// create new output group with the shared work queue
	clients := outGrp.Clients
	worker := make([]outputWorker, len(clients))
	for i, client := range clients {
		logger := logp.NewLogger("publisher_pipeline_output")
		worker[i] = makeClientWorker(c.observer, c.workQueue, client, logger, c.monitors.Tracer)
	}
	grp := &outputGroup{
		workQueue:  c.workQueue,
		outputs:    worker,
		timeToLive: outGrp.Retry + 1,
		batchSize:  outGrp.BatchSize,
	}

	c.out = grp

	// Resume consumer targeting the new work queue
	c.consumer.setTarget(
		consumerTarget{
			ch:         c.workQueue,
			batchSize:  grp.batchSize,
			timeToLive: grp.timeToLive,
		})
}

// Reload the output
func (c *outputController) Reload(
	cfg *reload.ConfigWithMeta,
	outFactory func(outputs.Observer, common.ConfigNamespace) (outputs.Group, error),
) error {
	outCfg := common.ConfigNamespace{}
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
