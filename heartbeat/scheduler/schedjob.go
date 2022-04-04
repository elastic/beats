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
	"context"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

type schedJob struct {
	id          string
	ctx         context.Context
	scheduler   *Scheduler
	wg          *sync.WaitGroup
	entrypoint  TaskFunc
	jobLimitSem *semaphore.Weighted
	activeTasks atomic.Int
}

// runRecursiveJob runs the entry point for a job, blocking until all subtasks are completed.
// Subtasks are run in separate goroutines.
// returns the time execution began on its first task
func newSchedJob(ctx context.Context, s *Scheduler, id string, jobType string, task TaskFunc) *schedJob {
	return &schedJob{
		id:          id,
		ctx:         ctx,
		scheduler:   s,
		jobLimitSem: s.jobLimitSem[jobType],
		entrypoint:  task,
		activeTasks: atomic.MakeInt(0),
		wg:          &sync.WaitGroup{},
	}
}

// runRecursiveTask runs an individual task and its continuations until none are left with as much parallelism as possible.
// Since task funcs can emit continuations recursively we need a function to execute
// recursively.
// The wait group passed into this function expects to already have its count incremented by one.
func (sj *schedJob) run() (startedAt time.Time) {
	sj.wg.Add(1)
	sj.activeTasks.Inc()
	if sj.jobLimitSem != nil {
		sj.jobLimitSem.Acquire(sj.ctx, 1)
	}

	startedAt = sj.runTask(sj.entrypoint)

	sj.wg.Wait()
	return startedAt
}

// runRecursiveTask runs an individual task and its continuations until none are left with as much parallelism as possible.
// Since task funcs can emit continuations recursively we need a function to execute
// recursively.
// The wait group passed into this function expects to already have its count incremented by one.
func (sj *schedJob) runTask(task TaskFunc) time.Time {
	defer sj.wg.Done()
	defer sj.activeTasks.Dec()

	// The accounting for waiting/active tasks is done using atomics.
	// Absolute accuracy is not critical here so the gap between modifying waitingTasks and activeJobs is acceptable.
	sj.scheduler.stats.waitingTasks.Inc()

	// Acquire an execution slot in keeping with heartbeat.scheduler.limit
	// this should block until resources are available.
	// In the case where the semaphore has free resources immediately
	// it will not block and will not check the cancelled status of the
	// context, which is OK, because we check it later anyway.
	limitErr := sj.scheduler.limitSem.Acquire(sj.ctx, 1)
	sj.scheduler.stats.waitingTasks.Dec()
	if limitErr == nil {
		defer sj.scheduler.limitSem.Release(1)
	}

	// Record the time this task started now that we have a resource to execute with
	startedAt := time.Now()

	// Check if the scheduler has been shut down. If so, exit early
	select {
	case <-sj.ctx.Done():
		return startedAt
	default:
		sj.scheduler.stats.activeTasks.Inc()

		continuations := task(sj.ctx)
		sj.scheduler.stats.activeTasks.Dec()

		sj.wg.Add(len(continuations))
		sj.activeTasks.Add(len(continuations))
		for _, cont := range continuations {
			// Run continuations in parallel, note that these each will acquire their own slots
			// We can discard the started at times for continuations as those are
			// irrelevant
			go sj.runTask(cont)
		}
		// There is always at least 1 task (the current one), if that's all, then we know
		// there are no other jobs active or pending, and we can release the jobLimitSem
		if sj.jobLimitSem != nil && sj.activeTasks.Load() == 1 {
			sj.jobLimitSem.Release(1)
		}
	}

	return startedAt
}
