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

package jobs

import (
	"testing"

	"github.com/menderesk/beats/v7/libbeat/beat"
)

// ExecJobsAndConts recursively executes multiple jobs.
func ExecJobsAndConts(t *testing.T, jobs []Job) ([]*beat.Event, error) {
	var results []*beat.Event
	for _, j := range jobs {
		resultEvents, err := ExecJobAndConts(t, j)
		if err != nil {
			return nil, err
		}
		for _, re := range resultEvents {
			results = append(results, re)
		}
	}

	return results, nil
}

// ExecJobAndConts will recursively execute a job and gather its results
func ExecJobAndConts(t *testing.T, j Job) ([]*beat.Event, error) {
	var results []*beat.Event
	event := &beat.Event{}
	results = append(results, event)
	cont, err := j(event)
	if err != nil {
		return nil, err
	}

	for _, cj := range cont {
		cjResults, err := ExecJobAndConts(t, cj)
		if err != nil {
			return nil, err
		}
		for _, cjResults := range cjResults {
			results = append(results, cjResults)
		}
	}

	return results, nil
}
