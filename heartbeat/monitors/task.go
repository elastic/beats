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
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// configuredJob represents a job combined with its config and any
// subsequent processors.
type configuredJob struct {
	job       jobs.Job
	config    jobConfig
	monitor   *Monitor
	cancelFn  context.CancelFunc
	pubClient pipeline.ISyncClient
}

func newConfiguredJob(job jobs.Job, config jobConfig, monitor *Monitor) *configuredJob {
	return &configuredJob{
		job:     job,
		config:  config,
		monitor: monitor,
	}
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

func (t *configuredJob) prepareSchedulerJob() scheduler.TaskFunc {
	return func(_ context.Context) []scheduler.TaskFunc {
		return runPublishJob(t.job, t.pubClient, NewJobState(2))
	}
}

// Start schedules this configuredJob for execution.
func (t *configuredJob) Start(pubClient pipeline.ISyncClient) {
	var err error

	t.pubClient = pubClient

	if err != nil {
		logp.L().Info("could not start monitor: %v", err)
		return
	}

	tf := t.prepareSchedulerJob()
	t.cancelFn, err = t.monitor.addTask(t.config.Schedule, t.monitor.stdFields.ID, tf, t.config.Type, pubClient.Wait)
	if err != nil {
		logp.L().Info("could not start monitor: %v", err)
	}
}

// Stop unschedules this configuredJob from execution.
func (t *configuredJob) Stop() {
	if t.cancelFn != nil {
		t.cancelFn()
	}
	if t.pubClient != nil {
		_ = t.pubClient.Close()
	}
}

type JobSummary struct {
	Attempt          uint32  `json:"attempt"`
	MaxAttempts      uint32  `json:"max_attempts"`
	FinalAttempt     bool    `json:"final_attempt"`
	Up               *uint32 `json:"up"`
	Down             *uint32 `json:"down"`
	Status           string  `json:"status"`
	mtx              sync.Mutex
	contsOutstanding *uint32
}

func NewJobState(maxAttempts uint32) *JobSummary {
	return &JobSummary{
		Attempt:          1,
		MaxAttempts:      maxAttempts,
		Up:               new(uint32),
		Down:             new(uint32),
		contsOutstanding: new(uint32),
		mtx:              sync.Mutex{},
	}
}

func runPublishJob(job jobs.Job, pubClient pipeline.ISyncClient, js *JobSummary) []scheduler.TaskFunc {
	event := &beat.Event{
		Fields: mapstr.M{},
	}

	conts, err := job(event)
	// Subtract one for the job we just ran, but add back in the length of the continuations
	outstandingConts := atomic.AddUint32(js.contsOutstanding, uint32(-1+len(conts)))
	if err != nil {
		logp.L().Info("Job failed with: %s", err)
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
			_ = pubClient.Publish(clone)
		} else {
			// no clone needed if no continuations
			_ = pubClient.Publish(*event)
		}
	}

	ms, err := event.GetValue("monitor.status")
	if err != nil {
		msStr, ok := ms.(string)
		if !ok {
			logp.L().Errorf("monitor status found, but wasn't a string: %v", ms)
		}

		if msStr == "up" {
			atomic.AddUint32(js.Up, 1)
		} else {
			atomic.AddUint32(js.Down, 1)
		}
	}

	// The job has completed, all continuations have executed
	if outstandingConts == 0 {
		// terminal event, should be a summary
		event.PutValue("summary", js)
	}

	contTasks := make([]scheduler.TaskFunc, len(conts))
	for i, cont := range conts {
		// Move the continuation into the local block scope
		// This is important since execution is deferred
		// Without this only the last continuation will be executed len(conts) times
		localCont := cont

		contTasks[i] = func(_ context.Context) []scheduler.TaskFunc {
			return runPublishJob(localCont, pubClient, js)
		}
	}
	return contTasks
}
