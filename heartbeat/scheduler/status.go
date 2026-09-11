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
	"time"

	"github.com/elastic/beats/v7/heartbeat/config"
)

// scheduleDelayThreshold is the smallest first-task start delay worth
// reporting. Healthy execution jitter is far below a second, so ignoring
// smaller delays lets an idle scheduler produce byte-identical snapshots that
// the management layer suppresses, instead of nudging a counter on every check.
// Live running/waiting gauges and delays at or above the threshold still
// publish news as they happen.
const scheduleDelayThreshold = time.Second

// ScheduleDelayStatus counts first-task starts that began at least
// scheduleDelayThreshold after their scheduled time. Starts that were on time
// leave every counter untouched.
type ScheduleDelayStatus struct {
	Count   uint64 `json:"count"`
	TotalMS uint64 `json:"total_ms"`
	// MaxMS is the largest delay seen over the whole process lifetime and is
	// never reset. Consumers looking for sustained pressure must compare
	// deltas of Count and TotalMS together with the current Waiting value,
	// because a single historical spike keeps MaxMS high forever.
	MaxMS uint64 `json:"max_ms"`
}

// JobTypeStatus reports scheduler pressure for a monitor type.
type JobTypeStatus struct {
	// Limit is the per-type concurrency limit, 0 meaning unlimited.
	Limit int64 `json:"limit"`
	// Running counts jobs holding the per-type semaphore.
	Running int64 `json:"running"`
	// Waiting counts jobs blocked on the per-type semaphore only.
	// Types without a per-type limit (http, tcp and icmp by default) therefore
	// always report 0 even under load; their contention with the global
	// scheduler limit surfaces through ScheduleDelay instead.
	Waiting int64 `json:"waiting"`
	// ScheduleDelay aggregates late first-task starts for this type.
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
	delayCount   atomic.Uint64
	delayTotalMS atomic.Uint64
	delayMaxMS   atomic.Uint64
}

// recordDelay samples how late a first task actually started. Delays shorter
// than scheduleDelayThreshold are dropped, including negative ones from clock
// adjustments, so a healthy scheduler never mutates these counters.
func (s *jobTypeStats) recordDelay(delay time.Duration) {
	if delay < scheduleDelayThreshold {
		return
	}
	delayMS := durationMillis(delay)

	s.delayCount.Add(1)
	s.delayTotalMS.Add(delayMS)
	for currentMax := s.delayMaxMS.Load(); delayMS > currentMax; currentMax = s.delayMaxMS.Load() {
		if s.delayMaxMS.CompareAndSwap(currentMax, delayMS) {
			break
		}
	}
}

func newJobTypeStats(jobLimitByType map[string]*config.JobLimit) map[string]*jobTypeStats {
	stats := make(map[string]*jobTypeStats, len(jobLimitByType))
	for jobType, jobLimit := range jobLimitByType {
		var limit int64
		if jobLimit != nil && jobLimit.Limit > 0 {
			limit = jobLimit.Limit
		}
		stats[jobType] = &jobTypeStats{limit: limit}
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
			ScheduleDelay: ScheduleDelayStatus{
				Count:   stats.delayCount.Load(),
				TotalMS: stats.delayTotalMS.Load(),
				MaxMS:   stats.delayMaxMS.Load(),
			},
		}
	}
	return Status{Jobs: jobs}
}

// durationMillis converts a non-negative duration to milliseconds. Negative
// values, including clock-adjustment artifacts, are reported as 0 so the
// conversion to uint64 cannot overflow.
func durationMillis(d time.Duration) uint64 {
	if d <= 0 {
		return 0
	}
	return uint64(d / time.Millisecond)
}
