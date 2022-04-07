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
	"github.com/elastic/beats/v8/libbeat/beat"
)

// A Job represents a unit of execution, and may return multiple continuation jobs.
type Job func(event *beat.Event) ([]Job, error)

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(f func(*beat.Event) error) Job {
	return func(event *beat.Event) ([]Job, error) {
		return nil, f(event)
	}
}

// JobWrapper is used for functions that wrap other jobs transforming their behavior.
type JobWrapper func(Job) Job

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

// JobWrapperFactory can be used to created new instances of JobWrappers.
type JobWrapperFactory func() JobWrapper

// WrapAllSeparately wraps the given jobs using the given JobWrapperFactory instances.
// This enables us to use a different JobWrapper for the jobs passed in, but recursively apply
// the same wrapper to their children.
func WrapAllSeparately(jobs []Job, factories ...JobWrapperFactory) []Job {
	var wrapped []Job
	for _, j := range jobs {
		for _, factory := range factories {
			wrapper := factory()
			j = Wrap(j, wrapper)
		}
		wrapped = append(wrapped, j)
	}
	return wrapped
}

// Wrap wraps the given Job and also any continuations with the given JobWrapper.
func Wrap(job Job, wrapper JobWrapper) Job {
	return func(event *beat.Event) ([]Job, error) {
		cont, err := wrapper(job)(event)
		return WrapAll(cont, wrapper), err
	}
}
