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

package scheduler

import (
	"sync/atomic"

	"github.com/elastic/beats/v7/heartbeat/config"
)

// ScheduleDelayStatus reports job scheduling delay measurements.
type ScheduleDelayStatus struct {
	Count   uint64 `json:"count"`
	TotalMS uint64 `json:"total_ms"`
	MaxMS   uint64 `json:"max_ms"`
}

// JobTypeStatus reports scheduler pressure for a monitor type.
type JobTypeStatus struct {
	Limit         int64               `json:"limit"`
	Running       int64               `json:"running"`
	Waiting       int64               `json:"waiting"`
	Runs          uint64              `json:"runs"`
	ScheduleDelay ScheduleDelayStatus `json:"schedule_delay"`
}

// Status is a snapshot of scheduler state.
type Status struct {
	Jobs map[string]JobTypeStatus `json:"jobs"`
}

type jobTypeStats struct {
	limit        int64
	running      atomic.Int64
	waiting      atomic.Int64
	runs         atomic.Uint64
	delayCount   atomic.Uint64
	delayTotalMS atomic.Uint64
	delayMaxMS   atomic.Uint64
}

func newJobTypeStats(jobLimitByType map[string]*config.JobLimit) map[string]*jobTypeStats {
	stats := make(map[string]*jobTypeStats, len(jobLimitByType))
	for jobType, jobLimit := range jobLimitByType {
		stats[jobType] = &jobTypeStats{limit: jobLimit.Limit}
	}
	return stats
}

func (s *Scheduler) getJobTypeStats(jobType string) *jobTypeStats {
	s.jobTypeStatsMu.Lock()
	defer s.jobTypeStatsMu.Unlock()

	stats, exists := s.jobTypeStats[jobType]
	if !exists {
		stats = &jobTypeStats{}
		s.jobTypeStats[jobType] = stats
	}
	return stats
}

// Status returns a snapshot of scheduler state.
func (s *Scheduler) Status() Status {
	s.jobTypeStatsMu.Lock()
	defer s.jobTypeStatsMu.Unlock()

	jobs := make(map[string]JobTypeStatus, len(s.jobTypeStats))
	for jobType, stats := range s.jobTypeStats {
		jobs[jobType] = JobTypeStatus{
			Limit:   stats.limit,
			Running: stats.running.Load(),
			Waiting: stats.waiting.Load(),
			Runs:    stats.runs.Load(),
			ScheduleDelay: ScheduleDelayStatus{
				Count:   stats.delayCount.Load(),
				TotalMS: stats.delayTotalMS.Load(),
				MaxMS:   stats.delayMaxMS.Load(),
			},
		}
	}
	return Status{Jobs: jobs}
}
