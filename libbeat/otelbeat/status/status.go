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

package status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/management/status"
)

type runnerState struct {
	state status.Status
	msg   string
}

type Reporter interface {
	GetReporterForRunner(id uint64) status.StatusReporter
}

func NewGroupStatusReporter(r status.StatusReporter) Reporter {
	if r == nil {
		return &nopStatus{}
	}
	return &reporter{
		reporter:     r,
		runnerStates: make(map[uint64]runnerState),
	}
}

type reporter struct {
	runnerStates map[uint64]runnerState
	reporter     status.StatusReporter
}

func (r *reporter) GetReporterForRunner(id uint64) status.StatusReporter {
	return &subReporter{
		id:     id,
		parent: r,
	}
}

func (r *reporter) updateStatusForRunner(id uint64, state status.Status, msg string) {
	if r.runnerStates == nil {
		r.runnerStates = make(map[uint64]runnerState)
	}

	// add status for the runner to the map
	r.runnerStates[id] = runnerState{
		state: state,
		msg:   msg,
	}

	// calculate the overall state of Metricbeat based on the module states
	calcState, calcMsg := r.calculateState()

	// report status to parent reporter
	r.reporter.UpdateStatus(calcState, calcMsg)
}

func (r *reporter) calculateState() (status.Status, string) {
	reportedState := status.Running
	reportedMsg := ""
	for _, s := range r.runnerStates {
		switch s.state {
		case status.Degraded:
			if reportedMsg != "" {
				// if multiple modules report degraded state, concatenate the messages
				reportedMsg = fmt.Sprintf("%s; %s", reportedMsg, s.msg)
			} else {
				reportedMsg = s.msg
			}
			reportedState = status.Degraded
		case status.Failed:
			// return the first failed runner
			return s.state, s.msg
		}
	}
	return reportedState, reportedMsg
}

type nopStatus struct{}

func (s *nopStatus) GetReporterForRunner(id uint64) status.StatusReporter {
	return nil
}

type subReporter struct {
	id     uint64
	parent *reporter
}

func (m *subReporter) UpdateStatus(status status.Status, msg string) {
	m.parent.updateStatusForRunner(m.id, status, msg)
}
