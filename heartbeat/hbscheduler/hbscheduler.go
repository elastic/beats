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

// Package hbscheduler shares Heartbeat schedulers between the Heartbeat
// instances running in a process.
//
// A Heartbeat process has exactly one Heartbeat instance, and therefore one
// scheduler, so `heartbeat.scheduler.limit` and the per job type
// `heartbeat.jobs.<type>.limit` settings bound the concurrency of every monitor
// it runs. That is no longer true when Heartbeat runs as an OTel receiver,
// because a process then hosts one Heartbeat instance per receiver and each of
// them would otherwise build its own scheduler, turning those settings from
// process-wide bounds into per receiver bounds.
//
// Acquire hands out reference counted schedulers grouped by an opaque key, so
// that instances which should share concurrency bounds also share a scheduler.
// Callers that want the standalone Heartbeat behavior pass an empty group.
package hbscheduler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// Params describes the scheduler a consumer asks for. Only the consumer that
// creates a group's scheduler gets to configure it; see Acquire.
type Params struct {
	// Limit is the maximum number of concurrently running tasks. Values below
	// one mean unlimited.
	Limit int64
	// Registry is where the scheduler publishes its metrics.
	Registry *monitoring.Registry
	// Location is the time zone schedules are evaluated in.
	Location *time.Location
	// JobLimitByType bounds concurrency per monitor type, e.g. `browser`.
	JobLimitByType map[string]*config.JobLimit
	// RunOnce puts the scheduler in run_once mode.
	RunOnce bool
}

// ReleaseFunc gives up a scheduler acquired with Acquire. It is safe to call
// more than once, and from multiple goroutines. A group's scheduler is stopped
// once every acquisition of it has been released.
type ReleaseFunc func()

type sharedScheduler struct {
	group  string
	sched  *scheduler.Scheduler
	params Params
	users  int
}

var (
	mtx sync.Mutex
	// schedulers holds the scheduler handed out for each group. Entries are
	// removed when their last consumer releases them.
	schedulers = map[string]*sharedScheduler{}
)

// Acquire returns the scheduler shared by the given group, creating it from
// params if the group does not have one yet. Later callers for the same group
// reuse its running scheduler; their params are only used to warn about
// settings that cannot be honored, with one exception: a run_once mismatch
// returns an error because mixing run_once and non-run_once consumers in a
// group causes WaitForRunOnce to behave incorrectly for both sides.
//
// The group is an opaque key. An empty group is the default group, which is
// what a standalone Heartbeat process uses.
//
// The caller must invoke the returned ReleaseFunc once it is done with the
// scheduler. A group's scheduler is stopped when its last consumer releases it.
func Acquire(logger *logp.Logger, group string, params Params) (*scheduler.Scheduler, ReleaseFunc, error) {
	logger = logger.Named("hbscheduler").With("scheduler_group", group)

	mtx.Lock()
	defer mtx.Unlock()

	// A stopped scheduler cannot be restarted, so it must not be handed out
	// again. This is defensive: a caller that invokes s.Stop() directly rather
	// than going through Release would leave a stopped scheduler in the map.
	if shared, ok := schedulers[group]; ok && shared.sched.Stopped() {
		logger.Debug("discarding stopped shared scheduler")
		delete(schedulers, group)
	}

	shared, ok := schedulers[group]
	if !ok {
		shared = &sharedScheduler{
			group: group,
			sched: scheduler.Create(
				params.Limit,
				params.Registry,
				params.Location,
				params.JobLimitByType,
				params.RunOnce,
				logger,
			),
			params: params,
		}
		schedulers[group] = shared
	} else {
		if shared.params.RunOnce != params.RunOnce {
			return nil, nil, fmt.Errorf(
				"scheduler group %q already has run_once=%t; "+
					"a consumer with run_once=%t cannot join it — "+
					"assign receivers with different run_once settings to separate scheduler groups",
				group, shared.params.RunOnce, params.RunOnce,
			)
		}
		if diffs := paramsDiff(shared.params, params); len(diffs) > 0 {
			logger.Warnf(
				"reusing the scheduler already running for this group, "+
					"ignoring conflicting scheduler settings: %s",
				strings.Join(diffs, ", "),
			)
		}
	}

	shared.users++
	logger.Debugf("acquired shared scheduler, consumers: %d", shared.users)

	var once sync.Once
	return shared.sched, func() {
		once.Do(func() { release(logger, shared) })
	}, nil
}

func release(logger *logp.Logger, shared *sharedScheduler) {
	mtx.Lock()
	defer mtx.Unlock()

	shared.users--
	if shared.users > 0 {
		logger.Debugf("released shared scheduler, consumers: %d", shared.users)
		return
	}

	// The scheduler this consumer held may already have been replaced, in which
	// case the group's new one must be left alone.
	if schedulers[shared.group] == shared {
		delete(schedulers, shared.group)
	}
	logger.Debug("released shared scheduler, stopping it as it has no consumers left")
	shared.sched.Stop()
}

// paramsDiff describes the settings in requested that the already running
// scheduler, configured with active, does not honor.
func paramsDiff(active, requested Params) []string {
	var diffs []string

	if limitName(active.Limit) != limitName(requested.Limit) {
		diffs = append(diffs, fmt.Sprintf("scheduler.limit: running with %s, requested %s", limitName(active.Limit), limitName(requested.Limit)))
	}
	if locationName(active.Location) != locationName(requested.Location) {
		diffs = append(diffs, fmt.Sprintf("scheduler.location: running with %s, requested %s", locationName(active.Location), locationName(requested.Location)))
	}
	for _, jobType := range jobTypes(active.JobLimitByType, requested.JobLimitByType) {
		activeLimit, requestedLimit := jobLimit(active.JobLimitByType, jobType), jobLimit(requested.JobLimitByType, jobType)
		if limitName(activeLimit) != limitName(requestedLimit) {
			diffs = append(diffs, fmt.Sprintf("jobs.%s.limit: running with %s, requested %s", jobType, limitName(activeLimit), limitName(requestedLimit)))
		}
	}

	return diffs
}

// limitName renders a concurrency limit, where anything below one is unlimited.
// Normalizing keeps equivalent limits, e.g. an unset and an explicit zero, from
// being reported as a conflict.
func limitName(limit int64) string {
	if limit < 1 {
		return "unlimited"
	}
	return strconv.FormatInt(limit, 10)
}

func locationName(location *time.Location) string {
	if location == nil {
		return "Local"
	}
	return location.String()
}

// jobLimit returns the limit configured for jobType, zero (unlimited) if there
// is none.
func jobLimit(jobLimitByType map[string]*config.JobLimit, jobType string) int64 {
	limit := jobLimitByType[jobType]
	if limit == nil {
		return 0
	}
	return limit.Limit
}

// jobTypes returns the sorted union of the job types in the given maps, so that
// diffs are reported in a stable order.
func jobTypes(maps ...map[string]*config.JobLimit) []string {
	seen := map[string]struct{}{}
	for _, m := range maps {
		for jobType := range m {
			seen[jobType] = struct{}{}
		}
	}

	types := make([]string, 0, len(seen))
	for jobType := range seen {
		types = append(types, jobType)
	}
	sort.Strings(types)
	return types
}
