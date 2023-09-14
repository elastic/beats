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
	"time"

	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// LightweightDurationPlugin handles the logic for writing the `monitor.duration.us` field
// for lightweight monitors.
type LightweightDurationPlugin struct {
	startedAt *time.Time
}

func (lwdsp *LightweightDurationPlugin) EachEvent(event *beat.Event, _ error) EachEventActions {
	// Effectively only runs once, on the first event
	if lwdsp.startedAt == nil {
		now := time.Now()
		lwdsp.startedAt = &now
	}
	return 0
}

func (lwdsp *LightweightDurationPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	_, _ = event.PutValue("monitor.duration.us", look.RTTMS(time.Since(*lwdsp.startedAt)))
	return 0
}

func (lwdsp *LightweightDurationPlugin) BeforeRetry() {}

// BrowserDurationPlugin handles the logic for writing the `monitor.duration.us` field
// for browser monitors.
type BrowserDurationPlugin struct {
	startedAt *time.Time
	endedAt   *time.Time
}

func (bwdsp *BrowserDurationPlugin) EachEvent(event *beat.Event, _ error) EachEventActions {
	et, _ := event.GetValue("synthetics.type")
	if et == "journey/start" {
		bwdsp.startedAt = &event.Timestamp
	} else if et == "journey/end" {
		bwdsp.endedAt = &event.Timestamp
	}

	return 0
}

func (bwdsp *BrowserDurationPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	// do not set the duration if we're missing data
	// usually because the runner failed in a weird way
	if bwdsp.startedAt == nil || bwdsp.endedAt == nil {
		return 0
	}

	_, _ = event.PutValue("monitor.duration.us", look.RTTMS(bwdsp.endedAt.Sub(*bwdsp.startedAt)))

	return 0
}

func (bwdsp *BrowserDurationPlugin) BeforeRetry() {}
