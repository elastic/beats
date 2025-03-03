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
	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/libbeat/beat"
)

type DropBrowserExtraEvents struct{}

func (d DropBrowserExtraEvents) EachEvent(event *beat.Event, _ error) EachEventActions {
	st := synthType(event)
	// Sending these events can break the kibana UI in various places
	// see: https://github.com/elastic/kibana/issues/166530
	if st == "cmd/status" {
		eventext.CancelEvent(event)
	}

	return 0
}

func (d DropBrowserExtraEvents) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	// noop
	return 0
}

func (d DropBrowserExtraEvents) BeforeRetry() {
	// noop
}

func (d DropBrowserExtraEvents) BeforeEachEvent(event *beat.Event) {
	// noop
}
