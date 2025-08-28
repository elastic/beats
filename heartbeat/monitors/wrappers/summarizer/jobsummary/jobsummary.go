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

package jobsummary

import (
	"fmt"

	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/monitorstate"
)

// JobSummary is the struct that is serialized in the `summary` field in the emitted event.
type JobSummary struct {
	Attempt      uint16                   `json:"attempt"`
	MaxAttempts  uint16                   `json:"max_attempts"`
	FinalAttempt bool                     `json:"final_attempt"`
	Up           uint16                   `json:"up"`
	Down         uint16                   `json:"down"`
	Status       monitorstate.StateStatus `json:"status"`
	RetryGroup   string                   `json:"retry_group"`
}

func NewJobSummary(attempt uint16, maxAttempts uint16, retryGroup string) *JobSummary {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	return &JobSummary{
		MaxAttempts: maxAttempts,
		Attempt:     attempt,
		RetryGroup:  retryGroup,
	}
}

// BumpAttempt swaps the JobSummary object's pointer for a new job summary
// that is a clone of the current one but with the Attempt field incremented.
func (js *JobSummary) BumpAttempt() {
	*js = *NewJobSummary(js.Attempt+1, js.MaxAttempts, js.RetryGroup)
}

func (js *JobSummary) String() string {
	return fmt.Sprintf("<JobSummary status=%s attempt=%d/%d, final=%t, up=%d/%d retryGroup=%s>", js.Status, js.Attempt, js.MaxAttempts, js.FinalAttempt, js.Up, js.Down, js.RetryGroup)
}
