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
type Job interface {
	Name() string
	Run(event *beat.Event) ([]Job, error)
}

// NamedJob represents a job with an explicitly specified name.
type NamedJob struct {
	name string
	run  func(event *beat.Event) ([]Job, error)
}

// CreateNamedJob makes a new NamedJob.
func CreateNamedJob(name string, run func(event *beat.Event) ([]Job, error)) *NamedJob {
	return &NamedJob{name, run}
}

// Name returns the name of this job.
func (f *NamedJob) Name() string {
	return f.name
}

// Run executes the job.
func (f *NamedJob) Run(event *beat.Event) ([]Job, error) {
	return f.run(event)
}

// AnonJob represents a job with no assigned name, backed by just a function.
type AnonJob func(event *beat.Event) ([]Job, error)

// Name returns "" for AnonJob values.
func (aj AnonJob) Name() string {
	return ""
}

// Run executes the function.
func (aj AnonJob) Run(event *beat.Event) ([]Job, error) {
	return aj(event)
}

// AfterJob creates a wrapped version of the given Job that runs additional
// code after the original Job, possibly altering return values.
func AfterJob(j Job, after func(*beat.Event, []Job, error) ([]Job, error)) Job {

	return CreateNamedJob(
		j.Name(),
		func(event *beat.Event) ([]Job, error) {
			next, err := j.Run(event)

			return after(event, next, err)
		},
	)
}

// AfterJobSuccess creates a wrapped version of the given Job that runs additional
// code after the original job if the original Job succeeds, possibly altering
// return values.
func AfterJobSuccess(j Job, after func(*beat.Event, []Job, error) ([]Job, error)) Job {
	return AfterJob(j, func(event *beat.Event, cont []Job, err error) ([]Job, error) {
		if err != nil {
			return cont, err
		}

		return after(event, cont, err)
	})
}

// MakeSimpleJob creates a new Job from a callback function. The callback should
// return an valid event and can not create any sub-tasks to be executed after
// completion.
func MakeSimpleJob(f func(*beat.Event) error) Job {
	return AnonJob(func(event *beat.Event) ([]Job, error) {
		err := f(event)
		return nil, err
	})
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
