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
	"context"
	"fmt"

	"github.com/elastic/beats/v8/heartbeat/eventext"
	"github.com/elastic/beats/v8/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v8/heartbeat/scheduler"
	"github.com/elastic/beats/v8/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

// configuredJob represents a job combined with its config and any
// subsequent processors.
type configuredJob struct {
	job      jobs.Job
	config   jobConfig
	monitor  *Monitor
	cancelFn context.CancelFunc
	client   *WrappedClient
}

func newConfiguredJob(job jobs.Job, config jobConfig, monitor *Monitor) (*configuredJob, error) {
	return &configuredJob{
		job:     job,
		config:  config,
		monitor: monitor,
	}, nil
}

// jobConfig represents fields needed to execute a single job.
type jobConfig struct {
	Name     string             `config:"pluginName"`
	Type     string             `config:"type"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`
}

// ProcessorsError is used to indicate situations when processors could not be loaded.
// This special type is used because these errors are caught and handled gracefully.
type ProcessorsError struct{ root error }

func (e ProcessorsError) Error() string {
	return fmt.Sprintf("could not load monitor processors: %s", e.root)
}

func (t *configuredJob) prepareSchedulerJob(job jobs.Job) scheduler.TaskFunc {
	return func(_ context.Context) []scheduler.TaskFunc {
		return runPublishJob(job, t.client)
	}
}

func (t *configuredJob) makeSchedulerTaskFunc() scheduler.TaskFunc {
	return t.prepareSchedulerJob(t.job)
}

// Start schedules this configuredJob for execution.
func (t *configuredJob) Start(client *WrappedClient) {
	var err error

	t.client = client

	if err != nil {
		logp.Err("could not start monitor: %v", err)
		return
	}

	tf := t.makeSchedulerTaskFunc()
	t.cancelFn, err = t.monitor.addTask(t.config.Schedule, t.monitor.stdFields.ID, tf, t.config.Type, client.wait)
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

func runPublishJob(job jobs.Job, client *WrappedClient) []scheduler.TaskFunc {
	event := &beat.Event{
		Fields: common.MapStr{},
	}

	conts, err := job(event)
	if err != nil {
		logp.Err("Job failed with: %s", err)
	}

	hasContinuations := len(conts) > 0

	if event.Fields != nil && !eventext.IsEventCancelled(event) {
		// If continuations are present we defensively publish a clone of the event
		// in the chance that the event shares underlying data with the events for continuations
		// This prevents races where the pipeline publish could accidentally alter multiple events.
		if hasContinuations {
			clone := beat.Event{
				Timestamp: event.Timestamp,
				Meta:      event.Meta.Clone(),
				Fields:    event.Fields.Clone(),
			}
			client.Publish(clone)
		} else {
			// no clone needed if no continuations
			client.Publish(*event)
		}
	}

	if !hasContinuations {
		return nil
	}

	contTasks := make([]scheduler.TaskFunc, len(conts))
	for i, cont := range conts {
		// Move the continuation into the local block scope
		// This is important since execution is deferred
		// Without this only the last continuation will be executed len(conts) times
		localCont := cont

		contTasks[i] = func(_ context.Context) []scheduler.TaskFunc {
			return runPublishJob(localCont, client)
		}
	}
	return contTasks
}
