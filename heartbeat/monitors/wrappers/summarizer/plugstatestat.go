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
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// StateStatusPlugin encapsulates the writing of the primary fields used by the summary,
// those being `state.*`, `status.*` , `event.type`, and `monitor.check_group`
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

func (ssp *StateStatusPlugin) EachEvent(event *beat.Event, _ error) EachEventActions {
	monitorStatus, err := event.GetValue("monitor.status")
	if err == nil && !eventext.IsEventCancelled(event) { // if this event contains a status...
		mss := monitorstate.StateStatus(monitorStatus.(string))

		if mss == monitorstate.StatusUp {
			ssp.js.Up++
		} else {
			ssp.js.Down++
		}
	}

	_, _ = event.PutValue("monitor.check_group", fmt.Sprintf("%s-%d", ssp.checkGroup, ssp.js.Attempt))

	return 0
}

func (ssp *StateStatusPlugin) OnSummary(event *beat.Event) OnSummaryActions {
	if ssp.js.Down > 0 {
		ssp.js.Status = monitorstate.StatusDown
	} else {
		ssp.js.Status = monitorstate.StatusUp
	}

	// Get the last status of this monitor, we use this later to
	// determine if a retry is needed
	lastStatus := ssp.stateTracker.GetCurrentStatus(ssp.sf)

	// FinalAttempt is true if no retries will occur
	retry := ssp.js.Status == monitorstate.StatusDown && ssp.js.Attempt < ssp.js.MaxAttempts
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
	if ssp.sf.Type == "browser" {
		fields["synthetics"] = mapstr.M{"type": "heartbeat/summary"}
	}
	eventext.MergeEventFields(event, fields)

	if retry {
		// mutate the js into the state for the next attempt
		ssp.js.BumpAttempt()
	}

	logp.L().Debugf("attempt info: %v == %v && %d < %d", ssp.js.Status, lastStatus, ssp.js.Attempt, ssp.js.MaxAttempts)

	if retry {
		return RetryOnSummary
	}
	return 0
}
