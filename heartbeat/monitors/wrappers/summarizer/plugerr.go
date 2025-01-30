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
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors/logger"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BrowserErrPlugins handles the logic for writing the `error` field
// for browser monitors, preferentially using the journey/end event's
// error field for errors.
type BrowserErrPlugin struct {
	summaryErrVal  interface{}
	summaryErr     error
	stepCount      int
	journeyEndRcvd bool
	attempt        int
}

func NewBrowserErrPlugin() *BrowserErrPlugin {
	return &BrowserErrPlugin{
		attempt: 1,
	}
}

func (esp *BrowserErrPlugin) BeforeEachEvent(event *beat.Event) {} // noop

func (esp *BrowserErrPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	// track these to determine if the journey
	// needs an error injected due to incompleteness
	st := synthType(event)
	switch st {
	case "step/end":
		esp.stepCount++
		// track step count for error logging
		// this is a bit of an awkward spot and combination of concerns, but it makes sense
		eventext.SetMeta(event, logger.META_STEP_COUNT, esp.stepCount)
	case "journey/end":
		esp.journeyEndRcvd = true
	}

	// Nothing else to do if there's no error
	if eventErr == nil {
		return 0
	}

	// Merge the error value into the event's "error" field
	errVal := errToFieldVal(eventErr)
	mergeErrVal(event, errVal)

	// If there is no error value OR this is the journey end event
	// record this as the definitive error
	if esp.summaryErrVal == nil || st == "journey/end" {
		esp.summaryErr = eventErr
		esp.summaryErrVal = errVal
	}

	return DropErrEvent
}

func (esp *BrowserErrPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	// If no journey end was received, make that the summary error
	if !esp.journeyEndRcvd {
		esp.summaryErr = fmt.Errorf("journey did not finish executing, %d steps ran (attempt: %d): %w", esp.stepCount, esp.attempt, esp.summaryErr)
		esp.summaryErrVal = errToFieldVal(esp.summaryErr)
	}

	if esp.summaryErrVal != nil {
		mergeErrVal(event, esp.summaryErrVal)
	}

	return 0
}

func (esp *BrowserErrPlugin) BeforeRetry() {
	attempt := esp.attempt + 1
	*esp = *NewBrowserErrPlugin()
	esp.attempt = attempt
}

// LightweightErrPlugin simply takes error return values
// and maps them into the "error" field in the event, return nil
// for all events thereafter
type LightweightErrPlugin struct{}

func NewLightweightErrPlugin() *LightweightErrPlugin {
	return &LightweightErrPlugin{}
}

func (esp *LightweightErrPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	if eventErr == nil {
		return 0
	}

	errVal := errToFieldVal(eventErr)
	mergeErrVal(event, errVal)

	return DropErrEvent
}

func (esp *LightweightErrPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	return 0
}

func (esp *LightweightErrPlugin) BeforeRetry() {
	// noop
}

func (esp *LightweightErrPlugin) BeforeEachEvent(event *beat.Event) {
	// noop
}

// errToFieldVal reflects on the error and returns either an *ecserr.ECSErr if possible, and a look.Reason otherwise
func errToFieldVal(eventErr error) (errVal interface{}) {
	var asECS *ecserr.ECSErr
	if errors.As(eventErr, &asECS) {
		// Override the message of the error in the event it was wrapped
		asECS.Message = eventErr.Error()
		errVal = asECS
	} else {
		errVal = look.Reason(eventErr)
	}
	return errVal
}

func mergeErrVal(event *beat.Event, errVal interface{}) {
	eventext.MergeEventFields(event, mapstr.M{"error": errVal})
}
