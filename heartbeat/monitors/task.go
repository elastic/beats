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

	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
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
	mj := NewMetaJob(t.job, t.monitor.stdFields, t.monitor.monitorStateTracker)
	return func(_ context.Context) []scheduler.TaskFunc {
		return mj.RunWithRetries(2)
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

type MetaJob struct {
	stdFields    stdfields.StdMonitorFields
	stateTracker *monitorstate.Tracker
	rootJob      jobs.Job
}

type MetaJobRun struct {
	checkGroup       string
	jobSummary       *JobSummary
	MetaJob          *MetaJob
	mtx              *sync.Mutex
	contsOutstanding uint16
}

func NewMetaJob(rootJob jobs.Job, sf stdfields.StdMonitorFields, st *monitorstate.Tracker) *MetaJob {
	return &MetaJob{
		rootJob:      rootJob,
		stateTracker: st,
		stdFields:    sf,
	}

}

func (mj *MetaJob) RunWithRetries(maxRetries uint16) []scheduler.TaskFunc {
	u, err := uuid.NewV1()
	if err != nil {
		panic(fmt.Sprintf("cannot generate UUIDs on this system: %s", err))
	}
	retryGroup := u.String()

	mjr := MetaJobRun{
		MetaJob:          mj,
		jobSummary:       NewJobSummary(1, maxRetries, retryGroup),
		contsOutstanding: 1,
		mtx:              &sync.Mutex{},
	}

	mjr.RunJob(mj.rootJob)
}

func (mjr *MetaJobRun) RunJob(j jobs.Job) []scheduler.TaskFunc {
	event := &beat.Event{
		Fields: mapstr.M{},
	}
	conts, err := j(event)

	if err != nil {
		logp.L().Infof("Job failed with: %s", err)
	}

	mjr.mtx.Lock()
	defer mjr.mtx.Unlock()

	mj := mjr.MetaJob
	js := mjr.jobSummary

	mjr.contsOutstanding--
	mjr.contsOutstanding += uint16(len(conts))

	ms, err := event.GetValue("monitor.status")
	if err == nil { // if this event contains a status...
		msStr, ok := ms.(string)
		if !ok {
			logp.L().Errorf("monitor status found, but wasn't a string: %v", ms)
		}

		if msStr == "up" {
			js.Up++
		} else {
			js.Down++
		}
	}

	// The job has completed, all continuations have executed
	if mjr.contsOutstanding == 0 {
		if js.Down > 0 {
			js.Status = monitorstate.StatusDown
		} else {
			js.Status = monitorstate.StatusUp
		}

		ms := mjr.MetaJob.stateTracker.RecordStatus(mj.stdFields, monitorstate.StateStatus(js.Status))

		// terminal event, should be a summary
		eventext.MergeEventFields(event, mapstr.M{
			"summary": js,
			"state":   ms,
		})

		// TODO: Clean up this equality
		stateChange := js.Status != mj.stateTracker.GetCurrentState(mj.stdFields).Status
		canRetry := js.Attempt < js.MaxAttempts

		logp.L().Infof("retry decision: %t && %t , state comp (%s == %s)", stateChange, canRetry, js.Status, mj.stateTracker.GetCurrentState(mj.stdFields).Status)
		if stateChange && canRetry {
			conts = []jobs.Job{mj.rootJob}
			retryMjr := *mjr
			retryMjr.jobSummary = NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
		} else {
			js.FinalAttempt = true
		}
	}

	return jobs.WrapAll(conts, func(cont jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			return mjr.RunJob(cont), nil
		}
	})
}

type JobSummary struct {
	Attempt      uint16                   `json:"attempt"`
	MaxAttempts  uint16                   `json:"max_attempts"`
	FinalAttempt bool                     `json:"final_attempt"`
	Up           uint16                   `json:"up"`
	Down         uint16                   `json:"down"`
	Status       monitorstate.StateStatus `json:"status"`
	RetryGroup   string                   `json:"retry_group"`
}

func NewJobSummary(attempt uint16, maxAttempts uint16, retryGroup string) *JobSummary {
	return &JobSummary{
		Attempt:     attempt,
		MaxAttempts: maxAttempts,
		RetryGroup:  retryGroup,
	}
}

func runPublishJob(job jobs.Job, pubClient pipeline.ISyncClient, mj *MetaJob) []scheduler.TaskFunc {
	conts, err := job(event)

	// Needed because we re-assign js on summary
	// TODO: refactor code to not do this weird mutation
	js.mtx.Lock()
	defer js.mtx.Unlock()

	var outstandingConts uint32

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

	contTasks := make([]scheduler.TaskFunc, len(conts))
	for i, cont := range conts {
		// Move the continuation into the local block scope
		// This is important since execution is deferred
		// Without this only the last continuation will be executed len(conts) times
		localCont := cont

		contTasks[i] = func(_ context.Context) []scheduler.TaskFunc {
			return runPublishJob(localCont, pubClient, contJs, sf, mst)
		}
	}
	return contTasks
}
