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
	"reflect"
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

// settings are the Params that shape the scheduler's behavior, in a comparable
// form. The registry is left out as it only matters to whoever creates it.
type settings struct {
	Limit     int64
	Location  string
	JobLimits map[string]int64
	RunOnce   bool
}

func (p Params) settings() settings {
	jobLimits := make(map[string]int64, len(p.JobLimitByType))
	for jobType, jobLimit := range p.JobLimitByType {
		if jobLimit != nil {
			jobLimits[jobType] = jobLimit.Limit
		}
	}
	location := "Local"
	if p.Location != nil {
		location = p.Location.String()
	}
	return settings{Limit: p.Limit, Location: location, JobLimits: jobLimits, RunOnce: p.RunOnce}
}

// ReleaseFunc gives up a scheduler acquired with Acquire. It is safe to call
// more than once, and from multiple goroutines. A group's scheduler is stopped
// once every acquisition of it has been released.
type ReleaseFunc func()

type sharedScheduler struct {
	group    string
	sched    *scheduler.Scheduler
	settings settings
	users    int
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
// settings that cannot be honored.
//
// The group is an opaque key. An empty group is the default group, which is
// what a standalone Heartbeat process, the only Heartbeat instance in its
// process, uses.
//
// The caller must invoke the returned ReleaseFunc once it is done with the
// scheduler. A group's scheduler is stopped when its last consumer releases it,
// and the next Acquire for the group creates a new one.
func Acquire(logger *logp.Logger, group string, params Params) (*scheduler.Scheduler, ReleaseFunc) {
	logger = logger.Named("hbscheduler").With("scheduler_group", group)

	mtx.Lock()
	defer mtx.Unlock()

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
			settings: params.settings(),
		}
		schedulers[group] = shared
	} else if requested := params.settings(); !reflect.DeepEqual(shared.settings, requested) {
		logger.Warnf(
			"reusing the scheduler already running for this group, ignoring its conflicting settings: running with %+v, requested %+v",
			shared.settings, requested,
		)
	}

	shared.users++
	logger.Debugf("acquired shared scheduler, consumers: %d", shared.users)

	var once sync.Once
	return shared.sched, func() {
		once.Do(func() { release(logger, shared) })
	}
}

func release(logger *logp.Logger, shared *sharedScheduler) {
	mtx.Lock()
	defer mtx.Unlock()

	shared.users--
	if shared.users > 0 {
		logger.Debugf("released shared scheduler, consumers: %d", shared.users)
		return
	}

	delete(schedulers, shared.group)
	logger.Debug("released shared scheduler, stopping it as it has no consumers left")
	shared.sched.Stop()
}
