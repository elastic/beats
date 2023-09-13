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

	"github.com/elastic/beats/v7/heartbeat/ecserr"
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BrowserErrPlugins handles the logic for writing the `error` field
// for browser monitors, preferentially using the journey/end event's
// error field for errors.
type BrowserErrPlugin struct {
	summaryErrVal interface{}
}

func (esp *BrowserErrPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	if eventErr == nil {
		return 0
	}

	errVal := errToFieldVal(eventErr)
	mergeErrVal(event, errVal)

	isJourneyEnd := false
	if synthType(event) == "journey/end" {
		isJourneyEnd = true
	}
	if esp.summaryErrVal == nil || isJourneyEnd {
		esp.summaryErrVal = errVal
	}

	return DropErrEvent
}

func (esp *BrowserErrPlugin) OnSummary(event *beat.Event) OnSummaryActions {
	if esp.summaryErrVal != nil {
		mergeErrVal(event, esp.summaryErrVal)
	}
	return 0
}

func (esp *BrowserErrPlugin) OnRetry() {
	esp.summaryErrVal = nil
}

// LightweightErrPlugin simply takes error return values
// and maps them into the "error" field in the event, return nil
// for all events thereafter
type LightweightErrPlugin struct{}

func (esp *LightweightErrPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	if eventErr == nil {
		return 0
	}

	errVal := errToFieldVal(eventErr)
	mergeErrVal(event, errVal)

	return DropErrEvent
}

func (esp *LightweightErrPlugin) OnSummary(event *beat.Event) OnSummaryActions {
	return 0
}

func (esp *LightweightErrPlugin) OnRetry() {
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

func synthType(event *beat.Event) string {
	synthType, err := event.GetValue("synthetics.type")
	if err != nil {
		return ""
	}

	str, ok := synthType.(string)
	if !ok {
		return ""
	}
	return str
}
