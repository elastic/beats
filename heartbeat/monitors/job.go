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

package monitors

import (
	"github.com/elastic/beats/libbeat/beat"
)

// A Job represents a unit of execution, and may return multiple continuation jobs.
type Job func(event *beat.Event) ([]Job, error)

// AfterJob creates a wrapped version of the given Job that runs additional
// code after the original Job, possibly altering return values.
func AfterJob(j Job, after func(*beat.Event, []Job, error) ([]Job, error)) Job {
	return func(event *beat.Event) ([]Job, error) {
		next, err := j(event)
		return after(event, next, err)
	}
}

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(f func(*beat.Event) error) Job {
	return func(event *beat.Event) ([]Job, error) {
		return nil, f(event)
	}
}

// WrapAll takes a list of jobs and wraps them all with the provided Job wrapping
// function.
func WrapAll(jobs []Job, fn func(Job) Job) []Job {
	var wrapped []Job
	for _, j := range jobs {
		wrapped = append(wrapped, fn(j))
	}
	return wrapped
}
