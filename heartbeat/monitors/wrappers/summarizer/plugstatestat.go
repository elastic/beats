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

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer/jobsummary"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// StateStatusPlugin encapsulates the writing of the primary fields used by the summary,
// those being `state.*`, `status.*` , `event.type`, and `monitor.check_group`
type BrowserStateStatusPlugin struct {
	cssp *commonSSP
}

func NewBrowserStateStatusplugin(stateTracker *monitorstate.Tracker, sf stdfields.StdMonitorFields) *BrowserStateStatusPlugin {
	return &BrowserStateStatusPlugin{
		cssp: newCommonSSP(stateTracker, sf),
	}
}

func (ssp *BrowserStateStatusPlugin) EachEvent(event *beat.Event, jobErr error) EachEventActions {
	if jobErr != nil {
		// Browser jobs only return either a single up or down
		// any err will mark it as a down job
		ssp.cssp.js.Down = 1
	}
	ssp.cssp.BeforeEach(event, jobErr)

	return 0
}

func (ssp *BrowserStateStatusPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	if ssp.cssp.js.Down == 0 {
		// Browsers don't have a prior increment of this, so set it to some
		// non-zero value
		ssp.cssp.js.Up = 1
	}

	res := ssp.cssp.BeforeSummary(event)
	// Browsers don't set this prior, so we set this here, as opposed to lightweight monitors
	_, _ = event.PutValue("monitor.status", string(ssp.cssp.js.Status))

	_, _ = event.PutValue("synthetics", mapstr.M{"type": "heartbeat/summary"})
	return res
}

func (ssp *BrowserStateStatusPlugin) BeforeRetry() {
	ssp.cssp.BeforeRetry()
}

func (ssp *BrowserStateStatusPlugin) BeforeEachEvent(event *beat.Event) {} //noop

// LightweightStateStatusPlugin encapsulates the writing of the primary fields used by the summary,
// those being `state.*`, `status.*` , `event.type`, and `monitor.check_group`
type LightweightStateStatusPlugin struct {
	cssp *commonSSP
}

func NewLightweightStateStatusPlugin(stateTracker *monitorstate.Tracker, sf stdfields.StdMonitorFields) *LightweightStateStatusPlugin {
	return &LightweightStateStatusPlugin{
		cssp: newCommonSSP(stateTracker, sf),
	}
}

func (ssp *LightweightStateStatusPlugin) EachEvent(event *beat.Event, jobErr error) EachEventActions {
	status := look.Status(jobErr)
	_, _ = event.PutValue("monitor.status", status)
	if !eventext.IsEventCancelled(event) { // if this event contains a status...
		mss := monitorstate.StateStatus(status)

		if mss == monitorstate.StatusUp {
			ssp.cssp.js.Up++
		} else {
			ssp.cssp.js.Down++
		}

	}

	ssp.cssp.BeforeEach(event, jobErr)

	return 0
}

func (ssp *LightweightStateStatusPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	return ssp.cssp.BeforeSummary(event)
}

func (ssp *LightweightStateStatusPlugin) BeforeRetry() {
	ssp.cssp.BeforeRetry()
}

func (ssp *LightweightStateStatusPlugin) BeforeEachEvent(event *beat.Event) {} // noop

type commonSSP struct {
	js           *jobsummary.JobSummary
	stateTracker *monitorstate.Tracker
	sf           stdfields.StdMonitorFields
	checkGroup   string
}

func newCommonSSP(stateTracker *monitorstate.Tracker, sf stdfields.StdMonitorFields) *commonSSP {
	uu, err := uuid.NewV1()
	if err != nil {
		logp.L().Errorf("could not create v1 UUID for retry group: %s", err)
	}
	js := jobsummary.NewJobSummary(1, sf.MaxAttempts, uu.String())
	return &commonSSP{
		js:           js,
		stateTracker: stateTracker,
		sf:           sf,
		checkGroup:   uu.String(),
	}
}

func (ssp *commonSSP) BeforeEach(event *beat.Event, err error) {
	_, _ = event.PutValue("monitor.check_group", fmt.Sprintf("%s-%d", ssp.checkGroup, ssp.js.Attempt))
}

func (ssp *commonSSP) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	if ssp.js.Down > 0 {
		ssp.js.Status = monitorstate.StatusDown
	} else {
		ssp.js.Status = monitorstate.StatusUp
	}

	// Get the last status of this monitor, we use this later to
	// determine if a retry is needed
	lastStatus := ssp.stateTracker.GetCurrentStatus(ssp.sf)

	curCheckDown := ssp.js.Status == monitorstate.StatusDown
	lastStateUpOrEmpty := lastStatus == monitorstate.StatusUp || lastStatus == monitorstate.StatusEmpty
	hasAttemptsRemaining := ssp.js.Attempt < ssp.js.MaxAttempts

	// retry if...
	retry := curCheckDown && // the current check is down
		lastStateUpOrEmpty && // we were previously up or had no previous state, if we were previously down we just check once
		hasAttemptsRemaining // and we are configured to actually make multiple attempts
	// if we aren't retrying this is the final attempt
	ssp.js.FinalAttempt = !retry

	ms := ssp.stateTracker.RecordStatus(ssp.sf, ssp.js.Status, ssp.js.FinalAttempt)

	// dereference the pointer since the pointer is pointed at the next step
	// after this
	jsCopy := *ssp.js

	fields := mapstr.M{
		"event":   mapstr.M{"type": "heartbeat/summary"},
		"summary": &jsCopy,
		"state":   ms,
	}

	eventext.MergeEventFields(event, fields)

	logp.L().Infof("attempt info: current(%v) == lastStatus(%v) && attempts(%d < %d)", ssp.js.Status, lastStatus, ssp.js.Attempt, ssp.js.MaxAttempts)

	if retry {
		return RetryBeforeSummary
	}

	return 0
}

func (ssp *commonSSP) BeforeRetry() {
	// mutate the js into the state for the next attempt
	ssp.js.BumpAttempt()
}
