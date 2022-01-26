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
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// WrapCommon applies the common wrappers that all monitor jobs get.
func WrapCommon(js []jobs.Job, stdMonFields stdfields.StdMonitorFields) []jobs.Job {
	if stdMonFields.Type == "browser" {
		return WrapBrowser(js, stdMonFields)
	} else {
		return WrapLightweight(js, stdMonFields)
	}
}

// WrapLightweight applies to http/tcp/icmp, everything but journeys involving node
func WrapLightweight(js []jobs.Job, stdMonFields stdfields.StdMonitorFields) []jobs.Job {
	return jobs.WrapAllSeparately(
		jobs.WrapAll(
			js,
			addMonitorMeta(stdMonFields, len(js) > 1),
			addMonitorStatus(stdMonFields.Type, false),
			addMonitorDuration,
		),
		func() jobs.JobWrapper {
			return makeAddSummary(stdMonFields.Type)
		})
}

// WrapBrowser is pretty minimal in terms of fields added. The browser monitor
// type handles most of the fields directly, since it runs multiple jobs in a single
// run it needs to take this task on in a unique way.
func WrapBrowser(js []jobs.Job, stdMonFields stdfields.StdMonitorFields) []jobs.Job {
	return jobs.WrapAll(
		js,
		addMonitorMeta(stdMonFields, len(js) > 1),
		addMonitorStatus(stdMonFields.Type, true),
	)
}

// addMonitorMeta adds the id, name, and type fields to the monitor.
func addMonitorMeta(stdMonFields stdfields.StdMonitorFields, isMulti bool) jobs.JobWrapper {
	return func(job jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, e := job(event)
			addMonitorMetaFields(event, time.Now(), stdMonFields, isMulti)
			return cont, e
		}
	}
}

func addMonitorMetaFields(event *beat.Event, started time.Time, sf stdfields.StdMonitorFields, isMulti bool) {
	id := sf.ID
	name := sf.Name

	// If multiple jobs are listed for this monitor, we can't have a single ID, so we hash the
	// unique URLs to create unique suffixes for the monitor.
	if isMulti {
		url, err := event.GetValue("url.full")
		if err != nil {
			logp.Error(errors.Wrap(err, "Mandatory url.full key missing!"))
			url = "n/a"
		}
		urlHash, _ := hashstructure.Hash(url, nil)
		id = fmt.Sprintf("%s-%x", sf.ID, urlHash)
	}

	fieldsToMerge := common.MapStr{
		"monitor": common.MapStr{
			"timespan": timespan(started, sf.Schedule, sf.Timeout),
		},
	}
	// Browser monitor fields are enriched separately
	// in the synthexec package - x-pack/heartbeat/monitors/browser/synthexec/enrich.go
	if sf.Type != "browser" {
		fieldsToMerge = common.MapStr{
			"monitor": common.MapStr{
				"id":       id,
				"name":     name,
				"type":     sf.Type,
				"timespan": timespan(started, sf.Schedule, sf.Timeout),
			},
		}
	}

	// Add service.name for APM interop
	if sf.Service.Name != "" {
		fieldsToMerge["service"] = common.MapStr{
			"name": sf.Service.Name,
		}
	}

	eventext.MergeEventFields(event, fieldsToMerge)
}

func timespan(started time.Time, sched *schedule.Schedule, timeout time.Duration) common.MapStr {
	maxEnd := sched.Next(started)

	if maxEnd.Sub(started) < timeout {
		maxEnd = started.Add(timeout)
	}

	return common.MapStr{
		"gte": started,
		"lt":  maxEnd,
	}
}

// addMonitorStatus wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func addMonitorStatus(monitorType string, summaryOnly bool) jobs.JobWrapper {
	return func(origJob jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, err := origJob(event)

			if summaryOnly {
				hasSummary, _ := event.Fields.HasKey("summary.up")
				if !hasSummary {
					return cont, nil
				}
			}

			fields := common.MapStr{
				"monitor": common.MapStr{
					"status": look.Status(err),
				},
			}
			if err != nil {
				fields["error"] = look.Reason(err)
			}
			eventext.MergeEventFields(event, fields)
			return cont, nil
		}
	}
}

// addMonitorDuration adds duration correctly for all non-browser jobs
func addMonitorDuration(job jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		start := time.Now()

		cont, err := job(event)

		if event != nil {
			eventext.MergeEventFields(event, common.MapStr{
				"monitor": common.MapStr{
					"duration": look.RTT(time.Since(start)),
				},
			})
			event.Timestamp = start
		}

		return cont, err
	}
}

// makeAddSummary summarizes the job, adding the `summary` field to the last event emitted.
func makeAddSummary(monitorType string) jobs.JobWrapper {
	// This is a tricky method. The way this works is that we track the state across jobs in the
	// state struct here.
	state := struct {
		mtx        sync.Mutex
		monitorId  string
		remaining  uint16
		up         uint16
		down       uint16
		checkGroup string
		generation uint64
	}{
		mtx: sync.Mutex{},
	}
	// Note this is not threadsafe, must be called from a mutex
	resetState := func() {
		state.remaining = 1
		state.up = 0
		state.down = 0
		state.generation++
		u, err := uuid.NewV1()
		if err != nil {
			panic(fmt.Sprintf("cannot generate UUIDs on this system: %s", err))
		}
		state.checkGroup = u.String()
	}
	resetState()

	return func(job jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, jobErr := job(event)
			state.mtx.Lock()
			defer state.mtx.Unlock()

			// If the event is cancelled we don't record it as being either up or down since
			// we discard the event anyway.
			var eventStatus interface{}
			if !eventext.IsEventCancelled(event) {
				// After each job
				eventStatus, _ = event.GetValue("monitor.status")
				if eventStatus == "up" {
					state.up++
				} else {
					state.down++
				}
			}

			event.PutValue("monitor.check_group", state.checkGroup)

			// Adjust the total remaining to account for new continuations
			state.remaining += uint16(len(cont))
			// Reduce total remaining to account for the just executed job
			state.remaining--

			// After last job
			if state.remaining == 0 {
				up := state.up
				down := state.down

				eventext.MergeEventFields(event, common.MapStr{
					"summary": common.MapStr{
						"up":   up,
						"down": down,
					},
				})
				resetState()
			}

			return cont, jobErr
		}
	}
}
