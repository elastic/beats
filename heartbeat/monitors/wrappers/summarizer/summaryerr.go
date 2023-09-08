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

// LightweightDurationSumPlugin handles the logic for writing the `monitor.duration.us` field
// for lightweight monitors.
type ErrSumPlugin struct {
	firstErrVal interface{}
}

func (esp *ErrSumPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	if eventErr == nil {
		return 0
	}

	var errVal interface{}
	var asECS *ecserr.ECSErr
	if errors.As(eventErr, &asECS) {
		// Override the message of the error in the event it was wrapped
		asECS.Message = eventErr.Error()
		errVal = asECS
	} else {
		errVal = look.Reason(eventErr)
	}
	mergeErrVal(event, errVal)

	return DropErrEvent
}

func (esp *ErrSumPlugin) OnSummary(event *beat.Event) OnSummaryActions {
	if esp.firstErrVal != nil {
		mergeErrVal(event, esp.firstErrVal)
	}
	return 0
}

func mergeErrVal(event *beat.Event, errVal interface{}) {
	eventext.MergeEventFields(event, mapstr.M{"error": errVal})
}
