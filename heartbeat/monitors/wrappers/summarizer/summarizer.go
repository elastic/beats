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

type SumPlugin interface {
	EachEvent(event *beat.Event)
	// If at least one plugin returns true a retry will be performed
	OnSummary(event *beat.Event) (doRetry bool)
}

type Summarizer struct {
	rootJob        jobs.Job
	contsRemaining uint16
	mtx            *sync.Mutex
	sf             stdfields.StdMonitorFields
	retryDelay     time.Duration
	plugins        []SumPlugin
	startedAt      time.Time
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
	plugins := make([]SumPlugin, 0, 2)
	if sf.Type == "browser" {
		plugins = append(plugins, &BrowserDurationSumPlugin{})
	} else {
		plugins = append(plugins, &LightweightDurationSumPlugin{})
	}
	plugins = append(plugins, NewStateStatusPlugin(mst, sf))

	return &Summarizer{
		rootJob:        rootJob,
		contsRemaining: 1,
		mtx:            &sync.Mutex{},
		sf:             sf,
		retryDelay:     time.Second,
		startedAt:      time.Now(),
		plugins:        plugins,
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

func (js *JobSummary) BumpAttempt() {
	*js = *NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
}

// Wrap wraps the given job in such a way that the last event summarizes all previous events
// and additionally adds some common fields like monitor.check_group to all events.
// This adds the state and summary top level fields.
func (s *Summarizer) Wrap(j jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		conts, jobErr := j(event)

		s.mtx.Lock()
		defer s.mtx.Unlock()

		s.contsRemaining-- // we just ran one cont, discount it
		// these many still need to be processed
		s.contsRemaining += uint16(len(conts))

		for _, plugin := range s.plugins {
			plugin.EachEvent(event)
		}

		if s.contsRemaining == 0 {
			var retry bool
			for _, plugin := range s.plugins {
				doRetry := plugin.OnSummary(event)
				if doRetry {
					retry = true
				}
			}

			if retry {
				// Bump the job summary for the next attempt
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

type StateStatusPlugin struct {
	js           *JobSummary
	stateTracker *monitorstate.Tracker
	sf           stdfields.StdMonitorFields
	checkGroup   string
}

func NewStateStatusPlugin(stateTracker *monitorstate.Tracker, sf stdfields.StdMonitorFields) *StateStatusPlugin {
	uu, err := uuid.NewV1()
	if err != nil {
		logp.L().Errorf("could not create v1 UUID for retry group: %s", err)
	}
	js := NewJobSummary(1, sf.MaxAttempts, uu.String())
	return &StateStatusPlugin{
		js:           js,
		stateTracker: stateTracker,
		sf:           sf,
		checkGroup:   uu.String(),
	}
}

func (ssp *StateStatusPlugin) EachEvent(event *beat.Event) {
	monitorStatus, err := event.GetValue("monitor.status")
	if err == nil && !eventext.IsEventCancelled(event) { // if this event contains a status...
		mss := monitorstate.StateStatus(monitorStatus.(string))

		if mss == monitorstate.StatusUp {
			ssp.js.Up++
		} else {
			ssp.js.Down++
		}
	}
}

func (ssp *StateStatusPlugin) OnSummary(event *beat.Event) (retry bool) {
	if ssp.js.Down > 0 {
		ssp.js.Status = monitorstate.StatusDown
	} else {
		ssp.js.Status = monitorstate.StatusUp
	}

	// Get the last status of this monitor, we use this later to
	// determine if a retry is needed
	lastStatus := ssp.stateTracker.GetCurrentStatus(ssp.sf)

	// FinalAttempt is true if no retries will occur
	retry = ssp.js.Status == monitorstate.StatusDown && ssp.js.Attempt < ssp.js.MaxAttempts
	ssp.js.FinalAttempt = !retry

	ms := ssp.stateTracker.RecordStatus(ssp.sf, ssp.js.Status, ssp.js.FinalAttempt)

	// dereference the pointer since the pointer is pointed at the next step
	// after this
	jsCopy := *ssp.js
	eventext.MergeEventFields(event, mapstr.M{
		"monitor.check_group": fmt.Sprintf("%s-%d", ssp.checkGroup, ssp.js.Attempt),
		"summary":             &jsCopy,
		"state":               ms,
	})

	if retry {
		// mutate the js into the state for the next attempt
		ssp.js.BumpAttempt()
	}

	logp.L().Debugf("attempt info: %v == %v && %d < %d", ssp.js.Status, lastStatus, ssp.js.Attempt, ssp.js.MaxAttempts)

	return !ssp.js.FinalAttempt
}
