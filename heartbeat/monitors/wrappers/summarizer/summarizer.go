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

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/logger"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// Summarizer produces summary events (with summary.* and other asssociated fields).
// It accumulates state as it processes the whole event field in order to produce
// this summary.
type Summarizer struct {
	rootJob        jobs.Job
	contsRemaining uint16
	mtx            *sync.Mutex
	sf             stdfields.StdMonitorFields
	mst            *monitorstate.Tracker
	retryDelay     time.Duration
	plugins        []SummarizerPlugin
	startedAt      time.Time
}

// EachEventActions is a set of options using bitmasks to inform execution after the EachEvent callback
type EachEventActions uint8

// DropErrEvent if will remove the error from the job return.
const DropErrEvent = 1

// OnSummaryActions is a set of options using bitmasks to inform execution after the OnSummary callback
type OnSummaryActions uint8

// RetryOnSummary will retry the job once complete.
const RetryOnSummary = 1

// SummarizerPlugin encapsulates functionality for the Summarizer that's easily expressed
// in one location. Prior to this code was strewn about a bit more and following it was
// a bit trickier.
type SummarizerPlugin interface {
	// EachEvent is called on each event, and allows for the mutation of events
	EachEvent(event *beat.Event, err error) EachEventActions
	// OnSummary is run on the final (summary) event for each monitor.
	OnSummary(event *beat.Event) OnSummaryActions
	// OnRetry is called before the first EachEvent in the event of a retry
	// can be used for resetting state between retries
	OnRetry()
}

// JobSummary is the struct that is serialized in the `summary` field in the emitted event.
type JobSummary struct {
	Attempt      uint16                   `json:"attempt"`
	MaxAttempts  uint16                   `json:"max_attempts"`
	FinalAttempt bool                     `json:"final_attempt"`
	Up           uint16                   `json:"up"`
	Down         uint16                   `json:"down"`
	Status       monitorstate.StateStatus `json:"status"`
	RetryGroup   string                   `json:"retry_group"`
}

func (js *JobSummary) String() string {
	return fmt.Sprintf("<JobSummary status=%s attempt=%d/%d, final=%t, up=%d/%d retryGroup=%s>", js.Status, js.Attempt, js.MaxAttempts, js.FinalAttempt, js.Up, js.Down, js.RetryGroup)
}

func NewSummarizer(rootJob jobs.Job, sf stdfields.StdMonitorFields, mst *monitorstate.Tracker) *Summarizer {
	s := &Summarizer{
		rootJob:        rootJob,
		contsRemaining: 1,
		mtx:            &sync.Mutex{},
		mst:            mst,
		sf:             sf,
		retryDelay:     time.Second,
		startedAt:      time.Now(),
	}
	s.setupPlugins()
	return s
}

func (s *Summarizer) setupPlugins() {
	if s.sf.Type == "browser" {
		s.plugins = append(
			s.plugins,
			&BrowserDurationSumPlugin{},
			&BrowserURLSumPlugin{},
		)
	} else {
		s.plugins = append(s.plugins, &LightweightDurationSumPlugin{})
	}

	s.plugins = append(
		s.plugins,
		&ErrSumPlugin{},
		NewStateStatusPlugin(s.mst, s.sf),
	)
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

// BumpAttempt swaps the JobSummary object's pointer for a new job summary
// that is a clone of the current one but with the Attempt field incremented.
func (js *JobSummary) BumpAttempt() {
	*js = *NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
}

// Wrap wraps the given job in such a way that the last event summarizes all previous events
// and additionally adds some common fields like monitor.check_group to all events.
// This adds the state and summary top level fields.
func (s *Summarizer) Wrap(j jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		conts, eventErr := j(event)

		s.mtx.Lock()
		defer s.mtx.Unlock()

		s.contsRemaining-- // we just ran one cont, discount it
		// these many still need to be processed
		s.contsRemaining += uint16(len(conts))

		for _, plugin := range s.plugins {
			actions := plugin.EachEvent(event, eventErr)
			if actions&DropErrEvent != 0 {
				eventErr = nil
			}
		}

		if s.contsRemaining == 0 {
			var retry bool
			for _, plugin := range s.plugins {
				actions := plugin.OnSummary(event)
				if actions&RetryOnSummary != 0 {
					retry = true
				}

			}

			if !retry {
				// on final run emits a metric for the service when summary events are complete
				logger.LogRun(event)
			} else {
				// Bump the job summary for the next attempt
				s.contsRemaining = 1

				// Delay retries by 1s for two reasons:
				// 1. Since ES timestamps are millisecond resolution they can happen so fast
				//    that it's hard to tell the sequence in which jobs executed apart in our
				//    kibana queries
				// 2. If the site error is very short 1s gives it a tiny bit of time to recover
				delayedRootJob := jobs.Wrap(s.rootJob, func(j jobs.Job) jobs.Job {
					return func(event *beat.Event) ([]jobs.Job, error) {
						for _, p := range s.plugins {
							p.OnRetry()
						}
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

		return conts, eventErr
	}
}
