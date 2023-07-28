package summarizer

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/gofrs/uuid"
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

func NewSummarizer(rootJob jobs.Job, sf stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) *Summarizer {
	uu, err := uuid.NewV1()
	if err != nil {
		logp.L().Errorf("could not create v1 UUID for retry group: %s", err)
	}
	return &Summarizer{
		rootJob:        rootJob,
		contsRemaining: 1,
		mtx:            &sync.Mutex{},
		jobSummary:     NewJobSummary(1, maxAttempts, uu.String()),
		checkGroup:     uu.String(),
		stateTracker:   mst,
		sf:             sf,
	}
}

func NewJobSummary(attempt uint16, maxAttempts uint16, retryGroup string) *JobSummary {
	return &JobSummary{
		MaxAttempts: maxAttempts,
		Attempt:     attempt,
		RetryGroup:  retryGroup,
	}
}

func AddSummarizer(sf stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) jobs.JobWrapper {
	return jobs.WrapStateful[*Summarizer](func(rootJob jobs.Job) jobs.StatefulWrapper[*Summarizer] {
		return NewSummarizer(rootJob, sf, mst, maxAttempts)
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

			lastStatus := s.stateTracker.GetCurrentStatus(s.sf)
			ms := s.stateTracker.RecordStatus(s.sf, js.Status)
			eventext.MergeEventFields(event, mapstr.M{
				"summary": js,
				"state":   ms,
			})

			// Time to retry, perhaps
			logp.L().Debugf("retry info: %v == %v && %d < %d", js.Status, lastStatus, js.Attempt, js.MaxAttempts)
			if js.Status != lastStatus && js.Attempt < js.MaxAttempts {
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
