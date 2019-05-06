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

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// WrapCommon applies the common wrappers that all monitor jobs get.
func WrapCommon(js []jobs.Job, id string, name string, typ string) []jobs.Job {
	return jobs.WrapAllSeparately(
		jobs.WrapAll(
			js,
			addMonitorStatus,
			addMonitorDuration,
		), func() jobs.JobWrapper {
			return addMonitorMeta(id, name, typ, len(js) > 1)
		}, func() jobs.JobWrapper {
			return makeAddSummary()
		})
}

// addMonitorMeta adds the id, name, and type fields to the monitor.
func addMonitorMeta(id string, name string, typ string, isMulti bool) jobs.JobWrapper {
	return func(job jobs.Job) jobs.Job {
		return func(event *beat.Event) ([]jobs.Job, error) {
			cont, e := job(event)
			thisID := id

			if isMulti {
				url, err := event.GetValue("url.full")
				if err != nil {
					logp.Error(errors.Wrap(err, "Mandatory url.full key missing!"))
					url = "n/a"
				}
				urlHash, _ := hashstructure.Hash(url, nil)
				thisID = fmt.Sprintf("%s-%x", id, urlHash)
			}

			eventext.MergeEventFields(
				event,
				common.MapStr{
					"monitor": common.MapStr{
						"id":   thisID,
						"name": name,
						"type": typ,
					},
				},
			)

			return cont, e
		}
	}
}

// addMonitorStatus wraps the given Job's execution such that any error returned
// by the original Job will be set as a field. The original error will not be
// passed through as a return value. Errors may still be present but only if there
// is an actual error wrapping the error.
func addMonitorStatus(origJob jobs.Job) jobs.Job {
	return func(event *beat.Event) ([]jobs.Job, error) {
		cont, err := origJob(event)
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

// addMonitorDuration executes the given Job, checking the duration of its run.
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
func makeAddSummary() jobs.JobWrapper {
	// This is a tricky method. The way this works is that we track the state across jobs in the
	// state struct here.
	state := struct {
		mtx        sync.Mutex
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
			cont, err := job(event)
			state.mtx.Lock()
			defer state.mtx.Unlock()

			// If the event is cancelled we don't record it as being either up or down since
			// we discard the event anyway.
			if !eventext.IsEventCancelled(event) {
				// After each job
				eventStatus, _ := event.GetValue("monitor.status")
				if eventStatus == "up" {
					state.up++
				} else {
					state.down++
				}
			}

			// No error check needed here
			event.PutValue("monitor.check_group", state.checkGroup)

			// Adjust the total remaining to account for new continuations
			state.remaining += uint16(len(cont))
			// Reduce total remaining to account for the just executed job
			state.remaining--

			// After last job
			if state.remaining == 0 {
				eventext.MergeEventFields(event, common.MapStr{
					"summary": common.MapStr{
						"up":   state.up,
						"down": state.down,
					},
				})
				resetState()
			}

			return cont, err
		}
	}
}
