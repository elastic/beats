package status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/management/status"
)

type Reporter interface {
	GetReporterForRunner(id string) status.StatusReporter
}

type reporter struct {
	runnerStates map[string]runnerState
	reporter     status.StatusReporter
}

func (s *reporter) GetReporterForRunner(id string) status.StatusReporter {
	return &subReporter{
		id: id,
		s:  s,
	}
}

type runnerState struct {
	state status.Status
	msg   string
}

func NewGroupStatusReporter(r status.StatusReporter) Reporter {
	if r == nil {
		return &nopStatus{}
	}
	return &reporter{
		reporter:     r,
		runnerStates: make(map[string]runnerState),
	}
}

func (r *reporter) updateStatusForRunner(id string, state status.Status, msg string) {
	if r.runnerStates == nil {
		r.runnerStates = make(map[string]runnerState)
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

func (s *nopStatus) GetReporterForRunner(id string) status.StatusReporter {
	return nil
}

type subReporter struct {
	id string
	s  *reporter
}

func (m *subReporter) UpdateStatus(status status.Status, msg string) {
	m.s.updateStatusForRunner(m.id, status, msg)
}
