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

package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule/cron"
)

// Schedule defines an interface for getting the next scheduled runtime for a job
type Schedule interface {
	// Next returns the next runAt a scheduled event occurs after the given runAt
	Next(now time.Time) (next time.Time)
	Interval() time.Duration
	// Returns true if this schedule type should run once immediately before checking Next.
	// Cron tasks run at exact times so should set this to false.
	RunOnInit() bool
}

// intervalScheduler defines a schedule that runs at fixed intervals.
type intervalScheduler struct {
	interval time.Duration
}

// RunOnInit returns true for interval schedulers.
func (s intervalScheduler) RunOnInit() bool {
	return true
}

func Parse(in string, monitorId string) (Schedule, error) {
	every := "@every"

	// add '@every' keyword
	if strings.HasPrefix(in, every) {
		interval := strings.TrimSpace(in[len(every):])
		d, err := time.ParseDuration(interval)
		if err != nil {
			return nil, err
		}

		return intervalScheduler{d}, nil
	}

	// fallback on cron scheduler parsers
	s, err := cron.Parse(in)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func MustParse(in string, monitorId string) Schedule {
	sched, err := Parse(in, monitorId)
	if err != nil {
		panic(fmt.Sprintf("could not parse schedule '%s': %s", in, err))
	}
	return sched
}

func (s intervalScheduler) Next(t time.Time) time.Time {
	return t.Add(s.interval)
}

func (s intervalScheduler) Interval() time.Duration {
	return s.interval
}

type TimespanBounds struct {
	Gte time.Time
	Lt  time.Time
}

func Timespan(t time.Time, s Schedule) (ts TimespanBounds) {
	ts.Gte = t.Add(-time.Duration(t.UnixNano() % s.Interval().Nanoseconds()))
	ts.Lt = ts.Gte.Add(s.Interval())
	return ts
}
