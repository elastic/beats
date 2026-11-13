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

package secretstores

import (
	"sync"
	"sync/atomic"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// Redactor removes the secret values of one monitor from the events it
// publishes, as a processor of its pipeline client, and from its log lines.
// The monitor's plugin gives it the values of every run of each of its jobs.
type Redactor struct {
	// mu serializes set, readers load all without it.
	mu sync.Mutex
	// runs holds the Scrubber of the latest run of each job. Runs of a job
	// never overlap, so the events in flight all come from these runs, even
	// while the jobs of a monitor with several endpoints pick up a rotated
	// secret one after the other.
	runs []*Scrubber
	// all is the Scrubber of the values of all runs.
	all atomic.Pointer[Scrubber]
}

// NewRedactor returns a Redactor without values to remove.
func NewRedactor() *Redactor {
	return &Redactor{}
}

// set makes s the Scrubber of the latest run of the job.
func (r *Redactor) set(job int, s *Scrubber) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job >= len(r.runs) {
		r.runs = append(r.runs, make([]*Scrubber, job+1-len(r.runs))...)
	}
	if r.runs[job].equal(s) {
		return
	}
	r.runs[job] = s
	r.all.Store(mergeScrubbers(r.runs))
}

// scrub returns v without the secret values.
func (r *Redactor) scrub(v string) string {
	return r.all.Load().String(v)
}

// Run removes the secret values from the fields of event.
func (r *Redactor) Run(event *beat.Event) (*beat.Event, error) {
	r.all.Load().Event(event)
	return event, nil
}

func (r *Redactor) String() string {
	return "redact_secret_store_values"
}
