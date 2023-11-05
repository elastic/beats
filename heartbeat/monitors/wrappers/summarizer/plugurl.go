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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// BrowserURLPlugin handles the logic for writing the error.* fields
type BrowserURLPlugin struct {
	urlFields mapstr.M
}

func (busp *BrowserURLPlugin) EachEvent(event *beat.Event, eventErr error) EachEventActions {
	if len(busp.urlFields) == 0 {
		if urlFields, err := event.GetValue("url"); err == nil {
			if ufMap, ok := urlFields.(mapstr.M); ok {
				busp.urlFields = ufMap
			}
		}
	}
	return 0
}

func (busp *BrowserURLPlugin) BeforeSummary(event *beat.Event) BeforeSummaryActions {
	if busp.urlFields != nil {
		_, err := event.PutValue("url", busp.urlFields)
		if err != nil {
			logp.L().Errorf("could not set URL value for browser job: %s", err)
		}
	}
	return 0
}

func (busp *BrowserURLPlugin) BeforeRetry() {
	busp.urlFields = nil
}

func (busp *BrowserURLPlugin) BeforeEachEvent(event *beat.Event) {} //noop
