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
	"sync"
)

type runnerState struct {
	state Status
	msg   string
}

type RunnerReporter interface {
	GetReporterForRunner(id uint64) StatusReporter
}

func NewGroupStatusReporter(parent StatusReporter) RunnerReporter {
	if parent == nil {
		return &nopStatus{}
	}
	return &reporter{
		parent:       parent,
		runnerStates: make(map[uint64]runnerState),
	}
}

type reporter struct {
	runnerStates map[uint64]runnerState
	parent       StatusReporter
	mtx          sync.Mutex
}

func (r *reporter) GetReporterForRunner(id uint64) StatusReporter {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return &subReporter{
		id: id,
		r:  r,
	}
}

func (r *reporter) updateStatusForRunner(id uint64, state Status, msg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
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
	r.parent.UpdateStatus(calcState, calcMsg)
}

func (r *reporter) calculateState() (Status, string) {
	reportedState := Running
	reportedMsg := ""
	for _, s := range r.runnerStates {
		switch s.state {
		case Degraded:
			if reportedState != Degraded {
				reportedState = Degraded
				reportedMsg = s.msg
			}
		case Failed:
			// return the first failed runner
			return s.state, s.msg
		}
	}
	return reportedState, reportedMsg
}

type nopStatus struct{}

func (s *nopStatus) GetReporterForRunner(id uint64) StatusReporter {
	return nil
}

type subReporter struct {
	id uint64
	r  *reporter
}

func (m *subReporter) UpdateStatus(status Status, msg string) {
	m.r.updateStatusForRunner(m.id, status, msg)
}
