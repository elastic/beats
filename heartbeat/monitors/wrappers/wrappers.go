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
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/summarizer"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// WrapCommon applies the common wrappers that all monitor jobs get.
func WrapCommon(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, stateLoader monitorstate.StateLoader) []jobs.Job {
	mst := monitorstate.NewTracker(stateLoader, false)
	var wrapped []jobs.Job
	if stdMonFields.Type != "browser" || stdMonFields.BadConfig {
		wrapped = WrapLightweight(js, stdMonFields, mst)
	} else {
		wrapped = WrapBrowser(js, stdMonFields, mst)
	}
	// Wrap just the root jobs with the summarizer
	// The summarizer itself wraps the continuations in a stateful way
	for i, j := range wrapped {
		j := j
		wrapped[i] = func(event *beat.Event) ([]jobs.Job, error) {
			s := summarizer.NewSummarizer(j, stdMonFields, mst)
			return s.Wrap(j)(event)
		}
	}
	return wrapped
}

// WrapLightweight applies to http/tcp/icmp, everything but journeys involving node
func WrapLightweight(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, mst *monitorstate.Tracker) []jobs.Job {
	return jobs.WrapAll(
		js,
		addMonitorTimespan(stdMonFields),
		addServiceName(stdMonFields),
		addMonitorMeta(stdMonFields, len(js) > 1),
	)
}

// WrapBrowser is pretty minimal in terms of fields added. The browser monitor
// type handles most of the fields directly, since it runs multiple jobs in a single
// run it needs to take this task on in a unique way.
func WrapBrowser(js []jobs.Job, stdMonFields stdfields.StdMonitorFields, mst *monitorstate.Tracker) []jobs.Job {
	return jobs.WrapAll(
		js,
		addMonitorTimespan(stdMonFields),
		addServiceName(stdMonFields),
		addMonitorMeta(stdMonFields, false),
	)
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
