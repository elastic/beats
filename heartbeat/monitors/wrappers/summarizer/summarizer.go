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

package summarizer

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Summarizer struct {
	rootJob        jobs.Job
	contsRemaining uint16
	mtx            *sync.Mutex
	jobSummary     *JobSummary
	checkGroup     string
	stateTracker   *monitorstate.Tracker
	sf             stdfields.StdMonitorFields
	retryDelay     time.Duration
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

func NewSummarizer(rootJob jobs.Job, sf stdfields.StdMonitorFields, mst *monitorstate.Tracker) *Summarizer {
	uu, err := uuid.NewV1()
	if err != nil {
		logp.L().Errorf("could not create v1 UUID for retry group: %s", err)
	}
	return &Summarizer{
		rootJob:        rootJob,
		contsRemaining: 1,
		mtx:            &sync.Mutex{},
		jobSummary:     NewJobSummary(1, sf.MaxAttempts, uu.String()),
		checkGroup:     uu.String(),
		stateTracker:   mst,
		sf:             sf,
		// private property, but can be overridden in tests to speed them up
		retryDelay: time.Second,
	}
}

func NewJobSummary(attempt uint16, maxAttempts uint16, retryGroup string) *JobSummary {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	return &JobSummary{
		MaxAttempts: maxAttempts,
		Attempt:     attempt,
		RetryGroup:  retryGroup,
	}
}

// Wrap wraps the given job in such a way that the last event summarizes all previous events
// and additionally adds some common fields like monitor.check_group to all events.
// This adds the state and summary top level fields.
func (s *Summarizer) Wrap(j jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		conts, jobErr := j(event)

		_, _ = event.PutValue("monitor.check_group", fmt.Sprintf("%s-%d", s.checkGroup, s.jobSummary.Attempt))

		s.mtx.Lock()
		defer s.mtx.Unlock()

		js := s.jobSummary

		s.contsRemaining-- // we just ran one cont, discount it
		// these many still need to be processed
		s.contsRemaining += uint16(len(conts))

		monitorStatus, err := event.GetValue("monitor.status")
		if err == nil && !eventext.IsEventCancelled(event) { // if this event contains a status...
			mss := monitorstate.StateStatus(monitorStatus.(string))

			if mss == monitorstate.StatusUp {
				js.Up++
			} else {
				js.Down++
			}
		}

		if s.contsRemaining == 0 {
			if js.Down > 0 {
				js.Status = monitorstate.StatusDown
			} else {
				js.Status = monitorstate.StatusUp
			}

			// Get the last status of this monitor, we use this later to
			// determine if a retry is needed
			lastStatus := s.stateTracker.GetCurrentStatus(s.sf)

			// FinalAttempt is true if no retries will occur
			js.FinalAttempt = js.Status != monitorstate.StatusDown || js.Attempt >= js.MaxAttempts

			ms := s.stateTracker.RecordStatus(s.sf, js.Status, js.FinalAttempt)

			eventext.MergeEventFields(event, mapstr.M{
				"summary": js,
				"state":   ms,
			})

			logp.L().Debugf("attempt info: %v == %v && %d < %d", js.Status, lastStatus, js.Attempt, js.MaxAttempts)
			if !js.FinalAttempt {
				// Reset the job summary for the next attempt
				// We preserve `s` across attempts
				s.jobSummary = NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
				s.contsRemaining = 1

				// Delay retries by 1s for two reasons:
				// 1. Since ES timestamps are millisecond resolution they can happen so fast
				//    that it's hard to tell the sequence in which jobs executed apart in our
				//    kibana queries
				// 2. If the site error is very short 1s gives it a tiny bit of time to recover
				delayedRootJob := jobs.Wrap(s.rootJob, func(j jobs.Job) jobs.Job {
					return func(event *beat.Event) ([]jobs.Job, error) {
						time.Sleep(s.retryDelay)
						return j(event)
					}
				})
				conts = []jobs.Job{delayedRootJob}
			}
		}

		// Wrap downstream jobs using the same state object this lets us create new state
		// on the first job, but re-use that same object on continuations.
		for i, cont := range conts {
			conts[i] = s.Wrap(cont)
		}

		return conts, jobErr
	}
}
