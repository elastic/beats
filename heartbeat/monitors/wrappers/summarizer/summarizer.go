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

func AddSummarizer(sf stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) jobs.JobWrapper {
	return jobs.WrapStateful[*Summarizer](func(rootJob jobs.Job) jobs.StatefulWrapper[*Summarizer] {
		return NewSummarizer(rootJob, sf, mst)
	})
}

func (s *Summarizer) Wrap(j jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		conts, jobErr := j(event)

		_, _ = event.PutValue("monitor.check_group", s.checkGroup)

		s.mtx.Lock()
		defer s.mtx.Unlock()

		js := s.jobSummary

		s.contsRemaining-- // we just ran one cont, discount it
		// these many still need to be processed
		s.contsRemaining += uint16(len(conts))

		monitorStatus, err := event.GetValue("monitor.status")
		if err == nil && !eventext.IsEventCancelled(event) { // if this event contains a status...
			msss := monitorstate.StateStatus(monitorStatus.(string))

			if msss == monitorstate.StatusUp {
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

			// Time to retry, perhaps
			lastStatus := s.stateTracker.GetCurrentStatus(s.sf)
			isFinalAttempt := js.Status == lastStatus || js.Attempt >= js.MaxAttempts
			js.FinalAttempt = isFinalAttempt
			ms := s.stateTracker.RecordStatus(s.sf, js.Status, isFinalAttempt)
			eventext.MergeEventFields(event, mapstr.M{
				"summary": js,
				"state":   ms,
			})

			logp.L().Debugf("retry info: %v == %v && %d < %d", js.Status, lastStatus, js.Attempt, js.MaxAttempts)
			if !isFinalAttempt {
				// Reset the job summary for the next attempt
				s.jobSummary = NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
				s.contsRemaining++
				s.checkGroup = fmt.Sprintf("%s-%d", s.checkGroup, s.jobSummary.Attempt)
				return []jobs.Job{s.rootJob}, jobErr
			}
		}

		return conts, jobErr
	}
}
