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

package wrappers

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/logger"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// WrapCommon applies the common wrappers that all monitor jobs get.
func WrapCommon(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, stateLoader monitorstate.StateLoader, maxAttempts uint16) []jobs.Job {
	mst := monitorstate.NewTracker(stateLoader, false)
	if stdMonFields.Type == "browser" {
		return WrapBrowser(js, stdMonFields, mst, maxAttempts)
	} else {
		return WrapLightweight(js, stdMonFields, mst, maxAttempts)
	}
}

// WrapLightweight applies to http/tcp/icmp, everything but journeys involving node
func WrapLightweight(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) []jobs.Job {
	wrapped := jobs.WrapAllSeparately(
		jobs.WrapAll(
			js,
			addMonitorTimespan(stdMonFields),
			addServiceName(stdMonFields),
			addMonitorMeta(stdMonFields, len(js) > 1),
			addMonitorStatus(nil),
			addMonitorErr,
			addMonitorDuration,
			logMonitorRun(nil),
		),
	)
	var swrapped []jobs.Job
	for _, wj := range wrapped {
		swrapped = append(swrapped, addSummarizer(stdMonFields, mst, maxAttempts)(wj))
	}
	return swrapped
}

// WrapBrowser is pretty minimal in terms of fields added. The browser monitor
// type handles most of the fields directly, since it runs multiple jobs in a single
// run it needs to take this task on in a unique way.
func WrapBrowser(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) []jobs.Job {
	wrapped := jobs.WrapAll(
		js,
		addMonitorTimespan(stdMonFields),
		addServiceName(stdMonFields),
		addMonitorMeta(stdMonFields, false),
		addMonitorStatus(byEventType("heartbeat/summary")),
		addMonitorErr,
		logMonitorRun(byEventType("heartbeat/summary")),
	)
	var swrapped []jobs.Job
	for _, wj := range wrapped {
		swrapped = append(swrapped, addSummarizer(stdMonFields, mst, maxAttempts)(wj))
	}
	return swrapped
}

type Summarizer struct {
	rootJob        jobs.Job
	contsRemaining uint16
	mtx            *sync.Mutex
	jobSummary     *JobSummary
	checkGroup     string
	stateTracker   *monitorstate.Tracker
	sf             stdfields.StdMonitorFields
}

func addSummarizer(sf stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) jobs.JobWrapper {
	return jobs.WrapStateful[*Summarizer](func(rootJob jobs.Job) jobs.StatefulWrapper[*Summarizer] {
		return newSummarizer(rootJob, sf, mst, maxAttempts)
	})
}

func newSummarizer(rootJob jobs.Job, sf stdfields.StdMonitorFields, mst *monitorstate.Tracker, maxAttempts uint16) *Summarizer {
	uu, err := uuid.NewV1()
	if err != nil {
		logp.L().Errorf("could not create v1 UUID for retry group: %s", err)
	}
	return &Summarizer{
		rootJob:        rootJob,
		contsRemaining: 1,
		mtx:            &sync.Mutex{},
		jobSummary:     newJobSummary(1, maxAttempts, uu.String()),
		checkGroup:     uu.String(),
		stateTracker:   mst,
		sf:             sf,
	}
}

func newJobSummary(attempt uint16, maxAttempts uint16, retryGroup string) *JobSummary {
	return &JobSummary{
		MaxAttempts: maxAttempts,
		Attempt:     attempt,
		RetryGroup:  retryGroup,
	}
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
			logp.L().Infof("RETRY INFO: %v == %v && %d < %d", js.Status, lastStatus, js.Attempt, js.MaxAttempts)
			if js.Status != lastStatus && js.Attempt < js.MaxAttempts {
				// Reset the job summary for the next attempt
				s.jobSummary = newJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
				s.contsRemaining++
				s.checkGroup = fmt.Sprintf("%s-%d", s.checkGroup, s.jobSummary.Attempt)
				return []jobs.Job{s.rootJob}, jobErr
			}
		}

		return conts, jobErr
	}
}

// addMonitorMeta adds the id, name, and type fields to the monitor.
func addMonitorMeta(sFields stdfields.StdMonitorFields, hashURLIntoID bool) jobs.JobWrapper {
	return func(job jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := job(event)

			id := sFields.ID
			name := sFields.Name
			// If multiple jobs are listed for this monitor, we can't have a single ID, so we hash the
			// unique URLs to create unique suffixes for the monitor.
			if hashURLIntoID {
				url, err := event.GetValue("url.full")
				if err != nil {
					logp.Error(fmt.Errorf("mandatory url.full key missing: %w", err))
					url = "n/a"
				}
				urlHash, _ := hashstructure.Hash(url, nil)
				id = fmt.Sprintf("%s-%x", sFields.ID, urlHash)
			}

			fields := mapstr.M{
				"type": sFields.Type,
			}

			// This should always be the default,
			// in case a browser monitor cannot be started due to validation errors
			fields["id"] = id
			fields["name"] = name

			if sFields.Origin != "" {
				fields["origin"] = sFields.Origin
			}

			eventext.MergeEventFields(event, mapstr.M{"monitor": fields})
			return cont, err
		}
	}
}

func addMonitorTimespan(sf stdfields.StdMonitorFields) jobs.JobWrapper {
	return func(origJob jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := origJob(event)

			eventext.MergeEventFields(event, mapstr.M{
				"monitor": mapstr.M{
					"timespan": timespan(time.Now(), sf.Schedule, sf.Timeout),
				},
			})
			return cont, err
		}
	}
}

// Add service.name to monitors for APM interop
func addServiceName(sf stdfields.StdMonitorFields) jobs.JobWrapper {
	return func(origJob jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := origJob(event)

			if sf.Service.Name != "" {
				eventext.MergeEventFields(event, mapstr.M{
					"service": mapstr.M{
						"name": sf.Service.Name,
					},
				})
			}
			return cont, err
		}
	}
}

func timespan(started time.Time, sched *schedule.Schedule, timeout time.Duration) mapstr.M {
	maxEnd := sched.Next(started)

	if maxEnd.Sub(started) < timeout {
		maxEnd = started.Add(timeout)
	}

	return mapstr.M{
		"gte": started,
		"lt":  maxEnd,
	}
}

// addMonitorStatus wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func addMonitorStatus(match EventMatcher) jobs.JobWrapper {
	return func(origJob jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := origJob(event)

			if match == nil || match(event) {
				eventext.MergeEventFields(event, mapstr.M{
					"monitor": mapstr.M{
						"status": look.Status(err),
					},
				})
			}

			return cont, err
		}
	}
}

func addMonitorErr(origJob jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		cont, err := origJob(event)

		if err != nil {
			var errVal interface{}
			var asECS *ecserr.ECSErr
			if errors.As(err, &asECS) {
				// Override the message of the error in the event it was wrapped
				asECS.Message = err.Error()
				errVal = asECS
			} else {
				errVal = look.Reason(err)
			}
			eventext.MergeEventFields(event, mapstr.M{"error": errVal})
		}

		return cont, nil
	}
}

// addMonitorDuration adds duration correctly for all non-browser jobs
func addMonitorDuration(job jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		start := time.Now()
		cont, err := job(event)
		duration := time.Since(start)

		if event != nil {
			eventext.MergeEventFields(event, mapstr.M{
				"monitor": mapstr.M{
					"duration": look.RTT(duration),
				},
			})
			event.Timestamp = start
		}

		return cont, err
	}
}

// logMonitorRun emits a metric for the service when summary events are complete.
func logMonitorRun(match EventMatcher) jobs.JobWrapper {
	return func(job jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := job(event)

			if match == nil || match(event) {
				logger.LogRun(event)
			}

			return cont, err
		}
	}
}

func byEventType(t string) func(event *beat.Event) bool {
	return func(event *beat.Event) bool {
		eventType, err := event.Fields.GetValue("event.type")
		if err != nil {
			return false
		}

		return eventType == t
	}
}

type EventMatcher func(event *beat.Event) bool
