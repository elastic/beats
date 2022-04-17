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

	"github.com/menderesk/beats/v7/heartbeat/scheduler"
	"github.com/menderesk/beats/v7/heartbeat/scheduler/schedule/cron"
)

type Schedule struct {
	scheduler.Schedule
}

// intervalScheduler defines a schedule that runs at fixed intervals.
type intervalScheduler struct {
	interval time.Duration
}

// RunOnInit returns true for interval schedulers.
func (s intervalScheduler) RunOnInit() bool {
	return true
}

func Parse(in string) (*Schedule, error) {
	every := "@every"

	// add '@every' keyword
	if strings.HasPrefix(in, every) {
		interval := strings.TrimSpace(in[len(every):])
		d, err := time.ParseDuration(interval)
		if err != nil {
			return nil, err
		}

		return &Schedule{intervalScheduler{d}}, nil
	}

	// fallback on cron scheduler parsers
	s, err := cron.Parse(in)
	if err != nil {
		return nil, err
	}
	return &Schedule{s}, nil
}

func MustParse(in string) *Schedule {
	sched, err := Parse(in)
	if err != nil {
		panic(fmt.Sprintf("could not parse schedule parsed with MustParse: %s", err))
	}
	return sched
}

func (s intervalScheduler) Next(t time.Time) time.Time {
	return t.Add(s.interval)
}

func (s *Schedule) Unpack(str string) error {
	tmp, err := Parse(str)
	if err == nil {
		*s = *tmp
	}
	return err
}
