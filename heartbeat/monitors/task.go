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

package monitors

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

type taskCanceller func() error

type task struct {
	job        Job
	config     taskConfig
	monitor    *Monitor
	processors *processors.Processors
	cancelFn   taskCanceller
	client     beat.Client
}

type taskConfig struct {
	Name     string             `config:"name"`
	Type     string             `config:"type"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`

	// Fields and tags to add to monitor.
	EventMetadata common.EventMetadata    `config:",inline"`
	Processors    processors.PluginConfig `config:"processors"`
}

// InvalidMonitorProcessorsError is used to indicate situations when processors could not be loaded.
// This special type is used because these errors are caught and handled gracefully.
type InvalidMonitorProcessorsError struct{ root error }

func (e InvalidMonitorProcessorsError) Error() string {
	return fmt.Sprintf("could not load monitor processors: %s", e.root)
}

func newTask(job Job, config taskConfig, monitor *Monitor) (*task, error) {
	t := &task{
		job:     job,
		config:  config,
		monitor: monitor,
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, InvalidMonitorProcessorsError{err}
	}
	t.processors = processors

	if err != nil {
		logp.Critical("Could not create client for monitor task %+v", t.monitor)
		return nil, errors.Wrap(err, "could not create client for monitor task")
	}

	return t, nil
}

func (t *task) prepareSchedulerJob(meta common.MapStr, run jobRunner) scheduler.TaskFunc {
	return func() []scheduler.TaskFunc {
		event, next, err := run()
		if err != nil {
			logp.Err("Job %v failed with: ", err)
		}

		if event.Fields != nil {
			event.Fields.DeepUpdate(meta)
			t.client.Publish(event)
		}

		if len(next) == 0 {
			return nil
		}

		continuations := make([]scheduler.TaskFunc, len(next))
		for i, n := range next {
			continuations[i] = t.prepareSchedulerJob(meta, n)
		}
		return continuations
	}
}

func (t *task) makeSchedulerTaskFunc() scheduler.TaskFunc {
	name := t.config.Name
	if name == "" {
		name = t.config.Type
	}

	meta := common.MapStr{
		"monitor": common.MapStr{
			"name": name,
			"type": t.config.Type,
		},
	}

	return t.prepareSchedulerJob(meta, t.job.Run)
}

// Start schedules this task for execution.
func (t *task) Start() {
	var err error
	t.client, err = t.monitor.pipelineConnector.ConnectWith(beat.ClientConfig{
		EventMetadata: t.config.EventMetadata,
		Processor:     t.processors,
	})
	if err != nil {
		logp.Err("could not start monitor: %v", err)
		return
	}

	tf := t.makeSchedulerTaskFunc()
	t.cancelFn, err = t.monitor.scheduler.Add(t.config.Schedule, t.job.Name(), tf)
	if err != nil {
		logp.Err("could not start monitor: %v, err")
	}
}

// Stop unschedules this task from execution.
func (t *task) Stop() {
	if t.cancelFn != nil {
		t.cancelFn()
	}
	if t.client != nil {
		t.client.Close()
	}
}
