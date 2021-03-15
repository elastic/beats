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
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// A Job represents a unit of execution, and may return multiple continuation jobs.
type Job func(event *beat.Event) ([]Job, reason.Reason)

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(f func(*beat.Event) reason.Reason) Job {
	return func(event *beat.Event) ([]Job, reason.Reason) {
		return nil, f(event)
	}
}

// JobWrapper is used for functions that wrap other jobs transforming their behavior.
type JobWrapper func(Job) Job

// JobWrapperFactory can be used to created new instances of JobWrappers.
type JobWrapperFactory func() JobWrapper

// WrapEachRun invokes the given factory once per schedule run, then applies the
// wrapper that factory produces recursively to any continuations. This is useful
// for when you want something to run once per 'root' job that creates a new wrapper,
// then re-use that wrapper for all subsequent jobs
func WrapEachRun(js []Job, wf ...JobWrapperFactory) []Job {
	var wrapped []Job
	for _, j := range js {
		j := j // store j for closure below
		wrapped = append(wrapped, func(event *beat.Event) ([]Job, reason.Reason) {
			wj := j
			for _, f := range wf {
				w := f()
				wj = Wrap(wj, w)
			}
			return wj(event)
		})
	}
	return wrapped
}

// Wrap wraps the given Job and also any continuations with the given JobWrapper.
func Wrap(job Job, wrapper JobWrapper) Job {
	return func(event *beat.Event) ([]Job, reason.Reason) {
		cont, err := wrapper(job)(event)
		return WrapAll(cont, wrapper), err
	}
}

// WrapAll wraps all jobs and their continuations with the given wrappers
func WrapAll(jobs []Job, wrappers ...JobWrapper) []Job {
	var wrapped []Job
	for _, j := range jobs {
		for _, wrapper := range wrappers {
			j = Wrap(j, wrapper)
		}
		wrapped = append(wrapped, j)
	}
	return wrapped
}
