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

	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/heartbeat/scheduler"
	"github.com/elastic/beats/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

type taskCanceller func() error

// configuredJob represents a job combined with its config and any
// subsequent processors.
type configuredJob struct {
	job        jobs.Job
	config     jobConfig
	monitor    *Monitor
	processors *processors.Processors
	cancelFn   taskCanceller
	client     beat.Client
}

func newConfiguredJob(job jobs.Job, config jobConfig, monitor *Monitor) (*configuredJob, error) {
	t := &configuredJob{
		job:     job,
		config:  config,
		monitor: monitor,
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, ProcessorsError{err}
	}
	t.processors = processors

	if err != nil {
		logp.Critical("Could not create client for monitor configuredJob %+v", t.monitor)
		return nil, errors.Wrap(err, "could not create client for monitor configuredJob")
	}

	return t, nil
}

// jobConfig represents fields needed to execute a single job.
type jobConfig struct {
	Name     string             `config:"pluginName"`
	Type     string             `config:"type"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`

	// Fields and tags to add to monitor.
	EventMetadata common.EventMetadata    `config:",inline"`
	Processors    processors.PluginConfig `config:"processors"`
}

// ProcessorsError is used to indicate situations when processors could not be loaded.
// This special type is used because these errors are caught and handled gracefully.
type ProcessorsError struct{ root error }

func (e ProcessorsError) Error() string {
	return fmt.Sprintf("could not load monitor processors: %s", e.root)
}

func (t *configuredJob) prepareSchedulerJob(job jobs.Job) scheduler.TaskFunc {
	return func() []scheduler.TaskFunc {
		event := &beat.Event{
			Fields: common.MapStr{},
		}
		next, err := job(event)
		hasContinuations := len(next) > 0

		if err != nil {
			logp.Err("Job %v failed with: ", err)
		}

		if event != nil && event.Fields != nil {
			// If continuations are present we defensively publish a clone of the event
			// in the chance that the event shares underlying data with the events for continuations
			// This prevents races where the pipeline publish could accidentally alter multiple events.
			if hasContinuations {
				clone := beat.Event{
					Timestamp: event.Timestamp,
					Meta:      event.Meta.Clone(),
					Fields:    event.Fields.Clone(),
				}
				t.client.Publish(clone)
			} else {
				// no clone needed if no continuations
				t.client.Publish(*event)
			}
		}

		if len(next) == 0 {
			return nil
		}

		continuations := make([]scheduler.TaskFunc, len(next))
		for i, n := range next {
			continuations[i] = t.prepareSchedulerJob(n)
		}
		return continuations
	}
}

func (t *configuredJob) makeSchedulerTaskFunc() scheduler.TaskFunc {
	return t.prepareSchedulerJob(t.job)
}

// Start schedules this configuredJob for execution.
func (t *configuredJob) Start() {
	var err error

	fields := common.MapStr{"event": common.MapStr{"dataset": "uptime"}}
	if t.monitor.factoryMetadata != nil {
		fields.DeepUpdate(t.monitor.factoryMetadata.Get())
	}

	t.client, err = t.monitor.pipelineConnector.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: t.config.EventMetadata,
			Processor:     t.processors,
			Fields:        fields,
		},
	})
	if err != nil {
		logp.Err("could not start monitor: %v", err)
		return
	}

	tf := t.makeSchedulerTaskFunc()
	t.cancelFn, err = t.monitor.scheduler.Add(t.config.Schedule, t.monitor.id, tf)
	if err != nil {
		logp.Err("could not start monitor: %v", err)
	}
}

// Stop unschedules this configuredJob from execution.
func (t *configuredJob) Stop() {
	if t.cancelFn != nil {
		t.cancelFn()
	}
	if t.client != nil {
		t.client.Close()
	}
}
